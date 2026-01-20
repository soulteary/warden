package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromFile_ValidFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-data.json")

	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test1@example.com"},
		{Phone: "13800138001", Mail: "test2@example.com"},
	}

	// Write test data
	data, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, data, 0o600)
	require.NoError(t, err)

	// Test reading
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
	// Create temporary file containing invalid JSON
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.json")

	invalidJSON := `{"invalid": json}`
	err := os.WriteFile(testFile, []byte(invalidJSON), 0o600)
	require.NoError(t, err)

	// Test reading invalid JSON
	result := FromFile(testFile)

	// Since JSON is invalid, should return empty slice (nil slice equals empty slice in Go)
	// According to implementation, json.Unmarshal returns empty slice on failure
	assert.Empty(t, result, "无效JSON应该返回空切片")
}

func TestFromFile_EmptyFile(t *testing.T) {
	// Create empty file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.json")

	err := os.WriteFile(testFile, []byte(""), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	// Empty file should return empty slice
	assert.Empty(t, result, "空文件应该返回空切片")
}

func TestFromFile_EmptyArray(t *testing.T) {
	// Create file containing empty array
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty-array.json")

	emptyArray := `[]`
	err := os.WriteFile(testFile, []byte(emptyArray), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Empty(t, result, "空数组应该返回空切片")
}

func TestFromFile_MalformedData(t *testing.T) {
	// Create malformed data file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "malformed.json")

	malformedData := `[{"phone": "123", "mail": "test@example.com"}, {"phone":}]`
	err := os.WriteFile(testFile, []byte(malformedData), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	// Malformed data should return empty slice (because JSON parsing failed)
	assert.Empty(t, result, "格式错误的数据应该返回空切片")
}

func TestFromFile_ValidSingleRecord(t *testing.T) {
	// Test single record
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
	// Test records with missing fields
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "missing-fields.json")

	// Only phone, no mail
	partialData := `[{"phone": "13800138000"}]`
	err := os.WriteFile(testFile, []byte(partialData), 0o600)
	require.NoError(t, err)

	result := FromFile(testFile)

	assert.Len(t, result, 1, "应该读取到1条记录")
	assert.Equal(t, "13800138000", result[0].Phone)
	assert.Empty(t, result[0].Mail, "Mail字段应该为空")
}

func TestFromFile_Unicode(t *testing.T) {
	// Test Unicode characters
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
