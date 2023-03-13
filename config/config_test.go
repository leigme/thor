package config

import "testing"

func TestConfig_Update(t *testing.T) {
	type fields struct {
		Port     string
		SaveDir  string
		FileExt  string
		FileSize string
		FileUnit string
	}
	type args struct {
		src map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test_config_update_1",
			fields: fields{
				Port:     "8080",
				SaveDir:  "test_config_update",
				FileExt:  "test_config_update|abc",
				FileSize: "3",
				FileUnit: "1024",
			},
			args: args{
				src: map[string]string{
					"p": "8090",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Port:     tt.fields.Port,
				SaveDir:  tt.fields.SaveDir,
				FileExt:  tt.fields.FileExt,
				FileSize: tt.fields.FileSize,
				FileUnit: tt.fields.FileUnit,
			}
			c.Update(tt.args.src)
		})
	}
}
