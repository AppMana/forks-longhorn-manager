package engineapi

import (
	"testing"

	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
)

func TestStrictLocalDataServerProtocol(t *testing.T) {
	replica := &longhorn.Replica{Spec: longhorn.ReplicaSpec{
		InstanceSpec: longhorn.InstanceSpec{VolumeName: "volume", VolumeSize: 1},
	}}
	_, args, err := getBinaryAndArgsForReplicaProcessCreation(
		replica, "C:/var/lib/longhorn/replicas/volume", "",
		longhorn.DataLocalityStrictLocal, DataServerProtocolNPIPE, DefaultReplicaPortCountV1, 12, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for index := 0; index+1 < len(args); index++ {
		if args[index] == "--data-server-protocol" && args[index+1] == DataServerProtocolNPIPE {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("replica arguments do not select %s: %v", DataServerProtocolNPIPE, args)
	}
}
