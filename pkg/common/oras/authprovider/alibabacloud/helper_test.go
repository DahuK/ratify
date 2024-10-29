package alibabacloud

import (
	"reflect"
	"testing"
)

func Test_getRegionFromArtifict(t *testing.T) {
	type args struct {
		artifact string
	}
	arg1 := "dahu-registry-vpc.cn-hangzhou.cr.aliyuncs.com"
	arg2 := "registry-vpc.cn-beijing.aliyuncs.com"
	arg3 := "test.bad"
	arg4 := "registry-vpc.cr.aliyuncs.com"

	tests := []struct {
		name    string
		args    args
		want    *AcrMetaInfo
		wantErr bool
	}{
		{"mock-test-get-region-from-artifict-1", args{arg1}, &AcrMetaInfo{
			InstanceName: "dahu",
			Region:       "cn-hangzhou",
		}, false},
		{"mock-test-get-region-from-artifict-2", args{arg2}, &AcrMetaInfo{
			Region: "cn-beijing",
		}, false},
		{"mock-test-get-region-from-artifict-3", args{arg3}, nil, true},
		{"mock-test-get-region-from-artifict-4", args{arg4}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRegionFromArtifict(tt.args.artifact)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRegionFromArtifict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRegionFromArtifict() got = %v, want %v", got, tt.want)
			}
		})
	}
}
