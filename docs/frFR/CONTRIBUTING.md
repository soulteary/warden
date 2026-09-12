# Guide de contribution

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Merci de l'intérêt que vous portez au projet Warden ! Toutes les formes de contribution sont les bienvenues.

## 📋 Table des matières

- [Comment contribuer](#comment-contribuer)
- [Mise en place de l'environnement de développement](#mise-en-place-de-lenvironnement-de-développement)
- [Normes de code](#normes-de-code)
- [Politique de traduction](#politique-de-traduction)
- [Normes de commit](#normes-de-commit)
- [Processus de pull request](#processus-de-pull-request)
- [Rapports de bogues et demandes de fonctionnalités](#rapports-de-bogues-et-demandes-de-fonctionnalités)

## 🚀 Comment contribuer

Vous pouvez contribuer des manières suivantes :

- **Signaler des bogues** : signalez les problèmes dans les GitHub Issues
- **Proposer des fonctionnalités** : proposez de nouvelles idées dans les GitHub Issues
- **Soumettre du code** : proposez des améliorations via des Pull Requests
- **Améliorer la documentation** : aidez à améliorer la documentation du projet
- **Répondre aux questions** : aidez les autres utilisateurs dans les Issues

En participant à ce projet, veuillez respecter tous les contributeurs, accepter la critique constructive et vous concentrer sur ce qui est le mieux pour le projet.

## 🛠️ Mise en place de l'environnement de développement

### Prérequis

- Go 1.27 ou version supérieure
- Redis (pour les tests)
- Git

### Démarrage rapide

```bash
# 1. Forker et cloner le projet
git clone https://github.com/your-username/warden.git
cd warden

# 2. Ajouter le dépôt amont
git remote add upstream https://github.com/soulteary/warden.git

# 3. Installer les dépendances
go mod download

# 4. Lancer les tests
go test ./...

# 5. Démarrer le service localement (vérifiez que Redis tourne)
go run .
```

## 📝 Normes de code

Veuillez respecter les normes de code suivantes :

1. **Suivre les normes officielles de Go** : [Effective Go](https://go.dev/doc/effective_go)
2. **Formater le code** : exécutez `go fmt ./...`
3. **Vérifier le code** : utilisez `golangci-lint` ou `go vet ./...`
4. **Écrire des tests** : les nouvelles fonctionnalités doivent inclure des tests
5. **Ajouter des commentaires** : les fonctions et types publics doivent avoir des commentaires de documentation
6. **Nommage des constantes** : toutes les constantes doivent utiliser le style `ALL_CAPS` (UPPER_SNAKE_CASE)

Pour les règles détaillées de style de code, consultez [CODE_STYLE.md](CODE_STYLE.md).

## 🌐 Politique de traduction

Warden fournit sa documentation et ses messages d'exécution en sept langues. Elles ne sont
**pas** toutes maintenues au même niveau d'exigence, et prétendre le contraire est
précisément la raison pour laquelle cinq locales ont pris silencieusement 18 clés de
traduction et plusieurs documents de retard.

**Niveaux**

| Niveau | Langues | Attente |
| --- | --- | --- |
| Faisant autorité | Anglais (`enUS`), chinois simplifié (`zhCN`) | Mises à jour dans la même pull request que la modification. Une PR qui change le comportement sans mettre les deux à jour est incomplète. |
| Au mieux | `deDE`, `frFR`, `itIT`, `jaJP`, `koKR` | Peuvent être en retard. Chaque document en retard porte une bannière renvoyant les lecteurs vers les versions faisant autorité. |

**Les chaînes d'exécution ne relèvent pas du « au mieux ».** `locales/*.json` est contrôlé
par `go test ./locales/`, qui échoue dès qu'une locale :

- omet une clé définie par `en.json` (ou en définit une qui n'y figure pas),
- présente une séquence de verbes `printf` (`%s`, `%d`) différente de la source anglaise, ou
- contient une valeur identique octet pour octet à l'anglaise (une chaîne non traduite).

Les clés manquantes retombent sur l'anglais à l'exécution, si bien que rien ne casse
visiblement — et c'est exactement pour cela que ce contrôle existe. Lorsque vous ajoutez un
message visible par l'utilisateur, ajoutez la clé aux **sept** fichiers de locale dans le
même commit. Si une valeur est légitimement identique à l'anglais (un emprunt, un nom de
protocole), ajoutez-la à `intentionallyIdentical` dans `locales/locales_test.go` avec un
commentaire plutôt que de supprimer le contrôle.

Exécutez `make docs-parity` pour voir à quel point chaque document traduit s'est éloigné de `enUS`.

## 📦 Normes de commit

### Format du message de commit

Nous utilisons la norme [Conventional Commits](https://www.conventionalcommits.org/) :

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat` : nouvelle fonctionnalité
- `fix` : correction de bogue
- `docs` : mise à jour de la documentation
- `style` : ajustement du formatage du code (sans incidence sur l'exécution)
- `refactor` : refactorisation du code
- `perf` : optimisation des performances
- `test` : relatif aux tests
- `chore` : modifications du processus de build ou des outils auxiliaires

### Exemples

```
feat(cache): Add Redis cache support

Implemented Redis-based distributed cache, supporting data persistence and multi-instance sharing.

Closes #123
```

```
fix(router): Fix pagination parameter validation issue

Fixed the issue where incorrect status code was returned when page_size exceeds maximum value.

Fixes #456
```

## 🔄 Processus de pull request

### Créer une pull request

```bash
# 1. Créer une branche de fonctionnalité
git checkout -b feature/your-feature-name

# 2. Apporter les modifications et committer
git add .
git commit -m "feat: Add new feature"

# 3. Synchroniser le code amont
git fetch upstream
git rebase upstream/main

# 4. Pousser la branche et créer la PR
git push origin feature/your-feature-name
```

### Liste de vérification de la pull request

Avant de soumettre une Pull Request, veuillez vous assurer que :

- [ ] Le code respecte les normes de code du projet
- [ ] Tous les tests passent (`go test ./...`)
- [ ] Le code est formaté (`go fmt ./...`)
- [ ] Les tests nécessaires ont été ajoutés
- [ ] La documentation associée a été mise à jour
- [ ] Le message de commit respecte les [normes de commit](#normes-de-commit)
- [ ] Le code passe les vérifications de lint

Toutes les Pull Requests font l'objet d'une revue de code. Merci de répondre rapidement aux commentaires de revue.

## 🐛 Rapports de bogues et demandes de fonctionnalités

Avant de créer une Issue, recherchez parmi les Issues existantes afin de vérifier que le problème ou la fonctionnalité n'a pas déjà été signalé.

### Modèle de rapport de bogue

```markdown
**Description**
Décrivez le bogue de façon claire et concise.

**Étapes de reproduction**
1. Exécuter '...'
2. Constater l'erreur

**Comportement attendu**
Décrivez de façon claire et concise ce que vous attendiez.

**Comportement observé**
Décrivez de façon claire et concise ce qui s'est réellement passé.

**Informations d'environnement**
- Système d'exploitation : [par ex. macOS 12.0]
- Version de Go : [par ex. 1.27]
- Version de Redis : [par ex. 7.0]
```

### Modèle de demande de fonctionnalité

```markdown
**Description de la fonctionnalité**
Décrivez de façon claire et concise la fonctionnalité souhaitée.

**Description du problème**
Quel problème cette fonctionnalité résout-elle ? Pourquoi est-elle nécessaire ?

**Solution proposée**
Décrivez de façon claire et concise comment vous envisagez sa mise en œuvre.
```

## 🎯 Pour commencer

Si vous souhaitez contribuer mais ne savez pas par où commencer, vous pouvez vous intéresser à :

- Les Issues portant l'étiquette `good first issue`
- Les Issues portant l'étiquette `help wanted`
- Les commentaires `TODO` dans le code
- Les améliorations de la documentation (corriger des fautes, clarifier, ajouter des exemples)

En cas de question, consultez les Issues et Pull Requests existantes, ou posez votre question dans l'Issue concernée.

---

Merci encore pour votre contribution au projet Warden ! 🎉
