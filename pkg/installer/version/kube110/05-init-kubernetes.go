package kube110

import (
	"fmt"
	"time"

	"github.com/kubermatic/kubeone/pkg/config"
	"github.com/kubermatic/kubeone/pkg/installer/util"
	"github.com/kubermatic/kubeone/pkg/ssh"
)

func initKubernetes(ctx *util.Context) error {
	ctx.Logger.Infoln("Initializing Kubernetes…")

	return util.RunTaskOnAllNodes(ctx, initKubernetesOnNode)
}

func initKubernetesOnNode(ctx *util.Context, node config.HostConfig, conn ssh.Connection) error {
	if err := kubeadmInit(ctx, conn); err != nil {
		return err
	}

	return waitForApiserver(ctx, node, conn)
}

func kubeadmInit(ctx *util.Context, conn ssh.Connection) error {
	ctx.Logger.Infoln("Running kubeadm…")

	_, _, _, err := util.RunShellCommand(conn, ctx.Verbose, `
sudo kubeadm init \
     --config=./{{ .WORK_DIR }}/cfg/master.yaml \
     --ignore-preflight-errors=Port-10250,FileAvailable--etc-kubernetes-manifests-etcd.yaml,FileExisting-crictl
`, util.TemplateVariables{
		"WORK_DIR": ctx.WorkDir,
	})

	return err
}

func waitForApiserver(ctx *util.Context, node config.HostConfig, conn ssh.Connection) error {
	var err error

	ctx.Logger.Infoln("Waiting for apiserver…")

	command := fmt.Sprintf(
		`curl --max-time 3 --fail --cacert /etc/kubernetes/pki/ca.crt https://%s:6443/healthz`,
		node.PublicAddress)

	for remaining := 20; remaining >= 0; remaining-- {
		_, _, _, err = util.RunCommand(conn, command, ctx.Verbose)
		if err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("Kubernetes apiserver did not come up, giving up")
	}

	return nil
}
