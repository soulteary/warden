"""Pure fake HTTP/inert command controls. No Go, Docker, ELF or network."""
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import yaml

spec=importlib.util.spec_from_file_location('recovery',Path(__file__).with_name('recover.py'))
r=importlib.util.module_from_spec(spec);spec.loader.exec_module(r)


class FakeClient:
    def __init__(self):
        self.calls=[];self.uploaded=[];self.published=False;self.lose_ack_at=None;self.override={}
        self.index=r.canonical({'schemaVersion':2,'manifests':[]})
    def request(self,method,url,raw=None,registry_token=None,limit=0):
        self.calls.append((method,url))
        if (method,url) in self.override:return self.override[(method,url)]
        if method=='POST':
            name=url.split('name=')[1]
            item=dict(id=100+len(self.uploaded),name=name,size=len(raw),digest='sha256:'+r.digest(raw),state='uploaded')
            self.uploaded.append(item)
            if len(self.uploaded)==self.lose_ack_at:raise TimeoutError('PRIVATE_ENDPOINT_TOKEN')
            return 201,r.canonical(item)
        if method=='PATCH':
            if r.decode(raw)!={'draft':False}:raise AssertionError('unexpected mutation')
            self.published=True
            return 200,r.canonical(dict(id=r.RELEASE,tag_name=r.TAG,draft=False))
        if url.startswith(r.REGISTRY):return 200,self.index
        if url.startswith('https://ghcr.io/token'):return 200,r.canonical({'token':'SYNTHETIC_PUBLIC'})
        path=url[len(r.API):]
        values={
          '/git/ref/tags/'+r.TAG:dict(ref='refs/tags/'+r.TAG,object=dict(sha=r.TAG_OBJECT,type='tag')),
          '/git/tags/'+r.TAG_OBJECT:dict(sha=r.TAG_OBJECT,tag=r.TAG,object=dict(sha=r.SOURCE,type='commit')),
          '/commits/'+r.SOURCE:dict(sha=r.SOURCE,commit=dict(tree=dict(sha=r.TREE))),
          '/actions/runs/'+str(r.RUN):dict(id=r.RUN,head_sha=r.SOURCE,event='push',head_branch=r.TAG,run_attempt=1,status='completed',conclusion='failure',path='.github/workflows/release.yml'),
          '/actions/runs/'+str(r.RUN)+'/jobs?per_page=100':dict(total_count=1,jobs=[dict(id=r.JOB,status='completed',conclusion='failure',steps=[dict(number=n,status='completed',conclusion='success')for n in (17,18,20,21,22)]+[dict(number=23,status='completed',conclusion='failure')])]),
          '/releases/'+str(r.RELEASE):dict(id=r.RELEASE,tag_name=r.TAG,draft=not self.published,prerelease=False,assets=self.uploaded.copy(),body='See CHANGELOG.md at '+r.SOURCE,author=dict(login='github-actions[bot]')),
          '/releases/'+str(r.RELEASE)+'/assets?per_page=100':self.uploaded.copy(),
        }
        if path=='/releases/tags/'+r.TAG:
            return (200,r.canonical({'id':r.RELEASE}))if self.published else(404,b'{"message":"Not Found"}')
        return 200,r.canonical(values[path])


class RecoveryTest(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.root=Path(self.temp.name).resolve();self.directory=self.root/'output';self.directory.mkdir(mode=0o700)
        self.tools=self.root/'tools';self.tools.mkdir(mode=0o700)
        self.client=FakeClient();self.events=[]
        self.addCleanup(patch.stopall)
        patch.object(r,'ROOT',self.tools).start()
        patch.object(r,'INDEX','sha256:'+r.digest(self.client.index)).start()
        patch.dict(os.environ,{'GITHUB_RUN_ID':'1000','GITHUB_RUN_ATTEMPT':'1','GITHUB_REPOSITORY':r.REPO,'GITHUB_EVENT_NAME':'workflow_dispatch','GITHUB_REF':'refs/heads/main','GITHUB_WORKFLOW_REF':r.REPO+'/.github/workflows/recover-warden163-owned-attempt.yml@refs/heads/main','GITHUB_WORKFLOW_SHA':'f'*40}).start()
        r._UPLOAD_STATE={'expected_assets':{},'known_uploaded_assets':[],'current_asset':None}
        expected=dict(schema='warden163-partial-image-qualification-v1',index_digest=r.INDEX,source_revision=r.SOURCE,tag_object=r.TAG_OBJECT,build_date=r.DATE,platforms=['linux/amd64','linux/arm64'],user='10001:10001',cosign_known_exit=0,source_constraints='exact-sha-ref-repository-push',wrong_source_control='specific-workflow-sha-mismatch',raw_graph_manifest_sha256='a'*64,cosign_capture_sha256='b'*64)
        q=r.canonical(expected);(self.tools/'image-qualification.json').write_bytes(q)
        self.qualification=r.digest(q)
        capsule=dict(schema='warden163-owned-recovery-capsule-v1',attempt=r.attempt(),image_qualification_sha256=self.qualification,recovery_nonce='c'*64)
        c=r.canonical(capsule);(self.tools/'capsule.json').write_bytes(c);self.capsule=r.digest(c)
        # Explicit synthetic qualification/pin collaborators, not real crypto.
        patch.object(r,'ROOT_IMAGE_REVIEW_SHA256',self.qualification).start()
        patch.object(r,'ROOT_CAPSULE_SHA256',self.capsule).start()
    def command(self,argv,stage,cwd,env):
        self.events.append((argv,stage,cwd,env));return 0
    def ready_assets(self):
        r.plan(self.directory,self.client,self.command,True)
        art=self.directory/'artifacts';art.mkdir(mode=0o700);hashes={}
        for name in r.BINARIES:
            raw=('synthetic-'+name).encode();(art/name).write_bytes(raw);hashes[name]=dict(size=len(raw),sha256=r.digest(raw))
        (art/'warden-image-sbom.spdx.json').write_bytes(b'{"spdxVersion":"SPDX-2.3"}')
        r.publish_json(self.directory,'rebuilt.json',dict(attempt=r.attempt(),original_unpublished_bytes_equal=False,binaries=hashes))
    def test_bound_plan_is_real_reads_and_exact_cosign_only(self):
        result=r.plan(self.directory,self.client,self.command,True)
        self.assertTrue(result['authority_bound']);self.assertTrue(all(v=='pass'for v in result['checks'].values()))
        self.assertTrue(all(m=='GET'for m,_ in self.client.calls));self.assertEqual(self.events[0][0],r.cosign_argv())
        self.assertIn('--certificate-github-workflow-sha',self.events[0][0])
    def test_missing_or_tampered_root_pins_never_build_or_write(self):
        with patch.object(r,'ROOT_CAPSULE_SHA256',None),self.assertRaises(r.Refused):r.rebuild(self.directory,self.root,self.client,self.command)
        self.assertEqual(self.events,[]);self.assertEqual(self.client.calls,[])
        raw=(self.tools/'capsule.json').read_bytes();(self.tools/'capsule.json').write_bytes(raw+b' ')
        with self.assertRaises(r.Refused):r.authority()
        self.assertEqual(self.events,[])
    def test_boolean_or_wrong_attempt_cannot_be_authority(self):
        value=r.decode((self.tools/'capsule.json').read_bytes());value['authorized']=True
        raw=r.canonical(value);(self.tools/'capsule.json').write_bytes(raw)
        with patch.object(r,'ROOT_CAPSULE_SHA256',r.digest(raw)),self.assertRaises(r.Refused):r.authority()
        del value['authorized'];value['attempt']['draft_id']=1;raw=r.canonical(value);(self.tools/'capsule.json').write_bytes(raw)
        with patch.object(r,'ROOT_CAPSULE_SHA256',r.digest(raw)),self.assertRaises(r.Refused):r.authority()
    def test_each_changed_live_binding_or_unknown_status_blocks(self):
        cases=[('/git/ref/tags/'+r.TAG,200,{'object':{'sha':'wrong'}}),('/commits/'+r.SOURCE,200,{'sha':'wrong'}),('/actions/runs/'+str(r.RUN),200,{'status':'in_progress'}),('/releases/'+str(r.RELEASE),200,{'id':r.RELEASE,'draft':False}),('/releases/'+str(r.RELEASE)+'/assets?per_page=100',200,[{'name':'foreign'}]),('/releases/tags/'+r.TAG,200,{'id':r.RELEASE}),('/releases/'+str(r.RELEASE),403,{'message':'PRIVATE'})]
        for path,status,body in cases:
            client=FakeClient();client.override[('GET',r.API+path)]=(status,r.canonical(body))
            with self.subTest(path=path),self.assertRaises(r.Refused):r.live_bindings(client,self.command)
            self.assertTrue(all(method=='GET'for method,_ in client.calls))
    def test_wrong_index_or_signature_is_not_accepted(self):
        self.client.index=b'wrong'
        with self.assertRaises(r.Refused):r.live_bindings(self.client,self.command)
        self.client.index=r.canonical({'schemaVersion':2,'manifests':[]})
        with self.assertRaises(r.Refused):r.live_bindings(self.client,lambda *a:1)
    def test_upload_only_fixed_id_and_eight_names_then_own_id_publish(self):
        self.ready_assets();result=r.upload(self.directory,self.client,self.command)
        self.assertTrue(result['published']);self.assertFalse(result['image_republished'])
        writes=[(m,u)for m,u in self.client.calls if m!='GET'];self.assertEqual(len(writes),9)
        self.assertTrue(all('/releases/'+str(r.RELEASE)+'/assets?name=' in url for method,url in writes[:-1]));self.assertEqual(writes[-1],('PATCH',r.API+'/releases/'+str(r.RELEASE)))
        self.assertEqual({a['name']for a in self.client.uploaded},set(r.FILES));self.assertTrue((self.directory/'published.json').exists())
        self.assertTrue(all('/releases/'+str(r.RELEASE) in u for _,u in writes))
    def test_lost_upload_ack_stops_and_preserves_unknown_without_retry(self):
        self.ready_assets();self.client.lose_ack_at=2
        with self.assertRaises(r.Refused)as caught:r.upload(self.directory,self.client,self.command)
        self.assertTrue(caught.exception.unknown);self.assertIsNone(caught.exception.status)
        self.assertEqual(len(self.client.uploaded),2);self.assertFalse(self.client.published)
        self.assertTrue((self.directory/'upload-start.json').exists());self.assertTrue((self.directory/('uploaded-'+r.FILES[0]+'.json')).exists())
        self.assertFalse((self.directory/'publish-intent.json').exists())
        before=len(self.client.calls)
        with self.assertRaises(Exception):r.upload(self.directory,self.client,self.command)
        self.assertEqual(len(self.client.calls),before)
    def test_partial_inspection_cannot_authorize_adoption_or_retry(self):
        self.client.uploaded=[dict(id=1,name=r.FILES[0],size=2,digest='sha256:'+'a'*64,state='uploaded')]
        result=r.inspect_partial(self.client);self.assertFalse(result['retry_authorized']);self.assertTrue(result['separate_partial_state_review_required'])
        with self.assertRaises(r.Refused):r.live_bindings(self.client,self.command)
        self.assertTrue(all(m=='GET'for m,_ in self.client.calls))
    def test_changed_build_record_or_extra_artifacts_blocks_before_post(self):
        self.ready_assets();(self.directory/'artifacts'/r.FILES[0]).write_bytes(b'changed')
        with self.assertRaises(r.Refused):r.upload(self.directory,self.client,self.command)
        self.assertTrue(all(m=='GET'for m,_ in self.client.calls))
    def test_rebuild_uses_exact_commands_with_inert_compiler(self):
        r.plan(self.directory,self.client,self.command,True)
        def build(argv,stage,cwd,env):
            self.events.append((argv,stage,cwd,env))
            if stage.startswith('go-build-'):Path(argv[argv.index('-o')+1]).write_bytes(stage.encode())
            return 0
        with patch.object(r,'source_checkout',return_value=self.root):result=r.rebuild(self.directory,self.root,self.client,build)
        self.assertEqual(result['binaries_rebuilt'],6);self.assertFalse(result['original_bytes_equality_claimed'])
        builds=[x for x in self.events if x[1].startswith('go-build-')];self.assertEqual(len(builds),6)
        for argv,stage,cwd,env in builds:self.assertIn(r.SOURCE,argv[argv.index('-ldflags')+1]);self.assertIn(r.DATE,argv[argv.index('-ldflags')+1]);self.assertEqual(env['CGO_ENABLED'],'0');self.assertNotIn('GH_TOKEN',env)
    def test_actual_temp_git_source_bytes_checked_despite_assume_unchanged(self):
        source=self.root/'source';source.mkdir()
        def git(*args):return subprocess.check_output(['git','-C',str(source),*args],stderr=subprocess.DEVNULL).decode().strip()
        git('init');(source/'plain.txt').write_text('public source')
        git('add','plain.txt');git('-c','user.name=Synthetic','-c','user.email=synthetic@example.invalid','-c','commit.gpgsign=false','commit','-m','fixture')
        sha=git('rev-parse','HEAD');tree=git('rev-parse','HEAD^{tree}')
        with patch.object(r,'SOURCE',sha),patch.object(r,'TREE',tree):
            self.assertEqual(r.source_checkout(source,self.command),source)
            git('update-index','--assume-unchanged','plain.txt');(source/'plain.txt').write_text('hidden change')
            with self.assertRaises(r.Refused):r.source_checkout(source,self.command)
            git('update-index','--no-assume-unchanged','plain.txt');(source/'plain.txt').write_text('public source')
            (source/'.git/info/exclude').write_text('hidden.txt\n');(source/'hidden.txt').write_text('not part of signed source')
            with self.assertRaises(r.Refused):r.source_checkout(source,self.command)
    def test_workflow_permissions_fixed_source_scanner_and_shell(self):
        root=Path(__file__).resolve().parents[2]
        workflow=yaml.safe_load((root/'.github/workflows/recover-warden163-owned-attempt.yml').read_text())
        self.assertFalse(workflow['concurrency']['cancel-in-progress'])
        for name,job in workflow['jobs'].items():
            self.assertNotIn('packages',job['permissions']);self.assertNotIn('id-token',job['permissions'])
            for step in job['steps']:
                if 'run'in step:subprocess.run(['bash','-n'],input=step['run'],text=True,capture_output=True,check=True)
        steps=workflow['jobs']['recover']['steps'];source=next(s for s in steps if s['name']=='Checkout exact original business source');self.assertEqual(source['with']['ref'],r.SOURCE)
        scanner=next(s for s in steps if 'sbom-action@'in s.get('uses',''))
        for option in ('dependency-snapshot','upload-artifact','upload-release-assets'):self.assertFalse(scanner['with'][option])
        self.assertIn(r.INDEX,scanner['with']['image']) if r.INDEX.startswith('sha256:9ac')else self.assertIn('sha256:9ac7db',scanner['with']['image'])
        self.assertEqual(scanner['with']['syft-version'],r.SYFT)
        self.assertEqual(scanner['env']['SYFT_PLATFORM'],'linux/amd64')
    def test_original_actions_run_reads_have_exact_job_permissions(self):
        root=Path(__file__).resolve().parents[2]
        workflow=yaml.safe_load((root/'.github/workflows/recover-warden163-owned-attempt.yml').read_text())
        self.assertEqual(workflow['jobs']['plan']['permissions'],{'actions':'read','contents':'write'})
        self.assertEqual(workflow['jobs']['recover']['permissions'],{'actions':'read','contents':'write'})
        r.live_bindings(self.client,self.command)
        paths=[url for method,url in self.client.calls if method=='GET']
        self.assertTrue(any('/actions/runs/'+str(r.RUN) in url for url in paths))
        self.assertTrue(any('/actions/runs/'+str(r.RUN)+'/jobs?per_page=100' in url for url in paths))
    def test_non_upload_clients_reject_mutation_before_opener(self):
        with patch.dict(os.environ,{'GH_TOKEN':'synthetic-canary'}):
            for phase in ('plan','inspect-partial','rebuild'):
                client=r.Client(phase)
                with patch.object(client.opener,'open') as opened:
                    for method,body in [('POST',b'{}'),('PATCH',b'{}'),('DELETE',None),('PUT',None),('GET',b'{}')]:
                        with self.assertRaises(r.Refused) as caught:
                            client.request(method,r.API+'/releases/'+str(r.RELEASE),body)
                        self.assertEqual(caught.exception.reason,'read-only-client')
                    opened.assert_not_called()
    def test_phase_selection_and_get_read_capability(self):
        class Response:
            code=200
            def read(self,limit):return b'{}'
            def __enter__(self):return self
            def __exit__(self,*args):pass
        with patch.dict(os.environ,{'GH_TOKEN':'synthetic-canary'}):
            for phase in ('plan','inspect-partial','rebuild','upload'):
                client=r.Client(phase)
                with patch.object(client.opener,'open',return_value=Response()) as opened:
                    self.assertEqual(client.request('GET',r.API+'/releases/'+str(r.RELEASE)),(200,b'{}'))
                    self.assertEqual(opened.call_args.args[0].get_method(),'GET')
            client=r.Client('upload')
            with patch.object(client.opener,'open',return_value=Response()) as opened:
                client.request('POST',r.API+'/releases/'+str(r.RELEASE),b'{}')
                self.assertEqual(opened.call_args.args[0].get_method(),'POST')
            with self.assertRaises(r.Refused):r.Client('unrecognized')
        source=Path(r.__file__).read_text()
        self.assertIn('Client(args.phase), Commands(directory, args.phase)',source)
    def test_real_inert_process_logs_are_separate_per_workflow_phase(self):
        # Actual Python child only, not Go/cosign/component execution.
        for phase in ('plan','rebuild','upload'):
            command=r.Commands(self.directory,phase)
            self.assertEqual(command([sys.executable,'-I','-B','-c','print("synthetic")'],'go-mod-verify',None,r.minimal_env()),0)
            self.assertEqual((self.directory/('logs-'+phase)/'1-go-mod-verify.private').read_text(),'synthetic\n')
    def test_non_manual_or_rerun_context_has_no_mutation_path(self):
        for changes in ({'GITHUB_EVENT_NAME':'push'},{'GITHUB_REF':'refs/heads/unreviewed'},{'GITHUB_RUN_ATTEMPT':'2'}):
            with patch.dict(os.environ,changes),self.assertRaises(r.Refused):r.plan(self.directory,self.client,self.command,True)
        self.assertTrue(all(m=='GET'for m,_ in self.client.calls))
    def test_atomic_roles_no_reuse_and_unknown_has_no_payload_message(self):
        r.publish_json(self.directory,'role.json',{'a':1})
        with self.assertRaises(r.Refused):r.publish_json(self.directory,'role.json',{'a':2})
        self.assertEqual(r.decode(r.read_file(self.directory/'role.json')),{'a':1})
        self.assertNotIn('PRIVATE',str(r.Refused('PRIVATE_ENDPOINT_TOKEN')))


if __name__=='__main__':unittest.main()
