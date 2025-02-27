package graterm

import (
	"testing"
)

func TestHook_String(t *testing.T) {
	t.Parallel()

	type fields struct {
		hook *Hook
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "nil_struct",
			fields: fields{
				hook: nil,
			},
			want: `<nil>`,
		},
		{
			name: "empty_non-nil_struct",
			fields: fields{
				hook: &Hook{},
			},
			want: `nameless component (order: 0)`,
		},
		{
			name: "nameless_hook",
			fields: fields{
				hook: &Hook{
					order: 3,
					name:  "   ",
				},
			},
			want: `nameless component (order: 3)`,
		},
		{
			name: "hook_with_a_name",
			fields: fields{
				hook: &Hook{
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

			got := tt.fields.hook.String()
			assertEqual(t, tt.want, got)
		})
	}
}
