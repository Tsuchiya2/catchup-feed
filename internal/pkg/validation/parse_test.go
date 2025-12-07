package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateISO8601(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantErr   bool
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "date only format - success",
			input:     "2024-01-01",
			wantNil:   false,
			wantErr:   false,
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   1,
		},
		{
			name:      "RFC3339 format with UTC - success",
			input:     "2024-01-01T10:00:00Z",
			wantNil:   false,
			wantErr:   false,
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   1,
		},
		{
			name:      "RFC3339 format with timezone - success",
			input:     "2024-12-25T15:30:00+09:00",
			wantNil:   false,
			wantErr:   false,
			wantYear:  2024,
			wantMonth: time.December,
			wantDay:   25,
		},
		{
			name:    "empty string - returns nil",
			input:   "",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "invalid format - slash separator",
			input:   "2024/01/01",
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "invalid format - no separator",
			input:   "20240101",
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "invalid date - month out of range",
			input:   "2024-13-01",
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "invalid date - day out of range",
			input:   "2024-01-32",
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "malformed - incomplete date",
			input:   "2024-01",
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateISO8601(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantYear, got.Year())
				assert.Equal(t, tt.wantMonth, got.Month())
				assert.Equal(t, tt.wantDay, got.Day())
			}
		})
	}
}

func TestParseDateISO8601_TimeComponent(t *testing.T) {
	// Test that time component is preserved in RFC3339 format
	input := "2024-01-01T10:30:45Z"
	got, err := ParseDateISO8601(input)

	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 10, got.Hour())
	assert.Equal(t, 30, got.Minute())
	assert.Equal(t, 45, got.Second())
}

func TestValidateEnum(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		allowed   []string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "valid value - first in list",
			value:     "RSS",
			allowed:   []string{"RSS", "Webflow"},
			fieldName: "source_type",
			wantErr:   false,
		},
		{
			name:      "valid value - last in list",
			value:     "Webflow",
			allowed:   []string{"RSS", "Webflow"},
			fieldName: "source_type",
			wantErr:   false,
		},
		{
			name:      "invalid value - not in list",
			value:     "Invalid",
			allowed:   []string{"RSS", "Webflow"},
			fieldName: "source_type",
			wantErr:   true,
		},
		{
			name:      "empty value - optional field",
			value:     "",
			allowed:   []string{"RSS", "Webflow"},
			fieldName: "source_type",
			wantErr:   false,
		},
		{
			name:      "case sensitive - lowercase not accepted",
			value:     "rss",
			allowed:   []string{"RSS"},
			fieldName: "source_type",
			wantErr:   true,
		},
		{
			name:      "single allowed value - match",
			value:     "RSS",
			allowed:   []string{"RSS"},
			fieldName: "source_type",
			wantErr:   false,
		},
		{
			name:      "empty allowed list - error",
			value:     "RSS",
			allowed:   []string{},
			fieldName: "source_type",
			wantErr:   true,
		},
		{
			name:      "nil allowed list - error",
			value:     "RSS",
			allowed:   nil,
			fieldName: "source_type",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnum(tt.value, tt.allowed, tt.fieldName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnum_ErrorMessage(t *testing.T) {
	// Test that error message includes field name and allowed values
	err := ValidateEnum("Invalid", []string{"RSS", "Webflow"}, "source_type")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_type")
	assert.Contains(t, err.Error(), "Invalid")
	assert.Contains(t, err.Error(), "RSS")
	assert.Contains(t, err.Error(), "Webflow")
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *bool
		wantErr bool
	}{
		{
			name:    "true string - lowercase",
			input:   "true",
			want:    boolPtr(true),
			wantErr: false,
		},
		{
			name:    "false string - lowercase",
			input:   "false",
			want:    boolPtr(false),
			wantErr: false,
		},
		{
			name:    "1 string - true",
			input:   "1",
			want:    boolPtr(true),
			wantErr: false,
		},
		{
			name:    "0 string - false",
			input:   "0",
			want:    boolPtr(false),
			wantErr: false,
		},
		{
			name:    "empty string - nil",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid string",
			input:   "invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "yes string - invalid",
			input:   "yes",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "no string - invalid",
			input:   "no",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBool(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)

			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

func TestParseBool_UpperCase(t *testing.T) {
	// Test uppercase variants (strconv.ParseBool accepts these)
	tests := []struct {
		input string
		want  bool
	}{
		{"TRUE", true},
		{"True", true},
		{"FALSE", false},
		{"False", false},
		{"T", true},
		{"t", true},
		{"F", false},
		{"f", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseBool(tt.input)

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

// boolPtr is a helper function to create a pointer to a boolean value
func boolPtr(b bool) *bool {
	return &b
}
