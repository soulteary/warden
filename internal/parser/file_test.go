package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestFromFile_ValidFile(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-data.json")

	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	// 写入测试数据
	data, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// 测试读取
	result := FromFile(testFile)

	assert.Len(t, result, 2, "应该读取到2条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "test1@example.com", result[0].Mail)
	assert.Equal(t, "13800138001", result[1].Phone)
	assert.Equal(t, "test2@example.com", result[1].Mail)
}

func TestFromFile_NonExistentFile(t *testing.T) {
	nonExistentFile := "/tmp/non-existent-file-12345.json"
	result := FromFile(nonExistentFile)

	assert.Empty(t, result, "不存在的文件应该返回空切片")
}

func TestFromFile_InvalidJSON(t *testing.T) {
	// 创建包含无效JSON的临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.json")

	invalidJSON := `{"invalid": json}`
	err := os.WriteFile(testFile, []byte(invalidJSON), 0o600)
	require.NoError(t, err)

	// 测试读取无效JSON
	result := FromFile(testFile)

	// 由于JSON无效，应该返回空切片（nil切片在Go中等于空切片）
	// 根据实现，json.Unmarshal失败时会返回空切片
	assert.Empty(t, result, "无效JSON应该返回空切片")
}

func TestFromFile_EmptyFile(t *testing.T) {
	// 创建空文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.json")

	err := os.WriteFile(testFile, []byte(""), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	// 空文件应该返回空切片
	assert.Empty(t, result, "空文件应该返回空切片")
}

func TestFromFile_EmptyArray(t *testing.T) {
	// 创建包含空数组的文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty-array.json")

	emptyArray := `[]`
	err := os.WriteFile(testFile, []byte(emptyArray), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Empty(t, result, "空数组应该返回空切片")
}

func TestFromFile_MalformedData(t *testing.T) {
	// 创建格式错误的数据文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "malformed.json")

	malformedData := `[{"phone": "123", "mail": "test@example.com"}, {"phone":}]`
	err := os.WriteFile(testFile, []byte(malformedData), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	// 格式错误的数据应该返回空切片（因为JSON解析失败）
	assert.Empty(t, result, "格式错误的数据应该返回空切片")
}

func TestFromFile_ValidSingleRecord(t *testing.T) {
	// 测试单个记录
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "single.json")

	singleRecord := []define.AllowListUser{
		{Phone: "13800138000", Mail: "single@example.com"},
	}

	data, err := json.Marshal(singleRecord)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Equal(t, "single@example.com", result[0].Mail)
}

func TestFromFile_MissingFields(t *testing.T) {
	// 测试缺少字段的记录
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "missing-fields.json")

	// 只有phone，没有mail
	partialData := `[{"phone": "13800138000"}]`
	err := os.WriteFile(testFile, []byte(partialData), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Empty(t, result[0].Mail, "Mail字段应该为空")
}

func TestFromFile_Unicode(t *testing.T) {
	// 测试Unicode字符
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unicode.json")

	unicodeData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "测试@example.com"},
	}

	data, err := json.Marshal(unicodeData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "测试@example.com", result[0].Mail, "应该正确处理Unicode字符")
}
