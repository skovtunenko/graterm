package graterm

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_terminationFunc_String(t *testing.T) {
	type fields struct {
		tf *Hook
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "nil_struct",
			fields: fields{
				tf: nil,
			},
			want: `<nil>`,
		},
		{
			name: "empty_non-nil_struct",
			fields: fields{
				tf: &Hook{},
			},
			want: `nameless component (order: 0)`,
		},
		{
			name: "nameless_termination_func",
			fields: fields{
				tf: &Hook{
					order: 3,
					name:  "   ",
				},
			},
			want: `nameless component (order: 3)`,
		},
		{
			name: "termination_function_with_a_name",
			fields: fields{
				tf: &Hook{
					order: 777,
					name:  "some random name",
				},
			},
			want: `component: "some random name" (order: 777)`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.fields.tf.String()
			require.Equal(t, tt.want, got)
		})
	}
}
