package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type errWriter struct{ n int }

func (e *errWriter) Write(p []byte) (int, error) {
	if e.n <= 0 {
		return 0, fmt.Errorf("write error")
	}
	e.n--
	return len(p), nil
}

func TestWriteJSON(t *testing.T) {
	vars := map[string]string{"Major": "1", "Minor": "2"}
	var buf bytes.Buffer
	err := WriteJSON(&buf, vars)
	require.NoError(t, err)

	var parsed map[string]string
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	require.Equal(t, "1", parsed["Major"])
	require.Equal(t, "2", parsed["Minor"])
}

func TestWriteVariable(t *testing.T) {
	vars := map[string]string{"SemVer": "1.2.3"}
	var buf bytes.Buffer
	err := WriteVariable(&buf, vars, "SemVer")
	require.NoError(t, err)
	require.Equal(t, "1.2.3\n", buf.String())
}

func TestWriteVariable_Unknown(t *testing.T) {
	vars := map[string]string{"SemVer": "1.2.3"}
	var buf bytes.Buffer
	err := WriteVariable(&buf, vars, "NonExistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown variable")
}

func TestWriteAll(t *testing.T) {
	vars := map[string]string{"A": "1", "B": "2"}
	var buf bytes.Buffer
	err := WriteAll(&buf, vars)
	require.NoError(t, err)
	require.Equal(t, "A=1\nB=2\n", buf.String())
}

func TestWriteJSON_WriteError(t *testing.T) {
	vars := map[string]string{"A": "1"}
	err := WriteJSON(&errWriter{n: 0}, vars)
	require.Error(t, err)
	require.Contains(t, err.Error(), "writing JSON output")
}

func TestWriteAll_WriteError(t *testing.T) {
	vars := map[string]string{"A": "1"}
	err := WriteAll(&errWriter{n: 0}, vars)
	require.Error(t, err)
}

func TestWriteAll_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteAll(&buf, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, "", buf.String())
}
