package util

import (
	"reflect"
	"testing"
)

func TestParseCommandLine(t *testing.T) {
	type args struct {
		command string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "value with space",
			args: args{
				command: "echo=\"hello world\" hi=this is a test",
			},
			want: []string{"echo=\"hello world\"", "hi=this", "is", "a", "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCommandLine(tt.args.command)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommandLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
