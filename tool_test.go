package main

import "testing"

func init() {
	testing.Init()
}

func TestFileMD5(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "test_FileMD5_1",
			args:    args{filePath: "/Users/leig/Documents/bt download.txt"},
			want:    "625cc763c4d01bd1ea7c1310d22d08e3",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FileMD5(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("FileMD5() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FileMD5() got = %v, want %v", got, tt.want)
			}
		})
	}
}
