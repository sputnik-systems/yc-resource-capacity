package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-sdk"
)

const (
	MemCapacity = 1024 * 1024 * 1024
)

type Resources struct {
	Cores  int64
	Memory float64
	Disks
}

type Disks struct {
	NetworkHDD float64
	NetworkSSD float64
}

func main() {
	folderID := flag.String("folder-id", "", "specify folder id")
	token := flag.String("token", "", "specify token")
	flag.Parse()

	ctx := context.Background()

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.OAuthToken(*token),
	})
	if err != nil {
		log.Fatalf("failed generate client: %s", err)
	}
	listInstancesResponse, err := sdk.Compute().Instance().List(ctx, &compute.ListInstancesRequest{
		FolderId: *folderID,
	})
	if err != nil {
		log.Fatalf("failed list instances: %s", err)
	}
	listDisksResponse, err := sdk.Compute().Disk().List(ctx, &compute.ListDisksRequest{
		FolderId: *folderID,
	})
	if err != nil {
		log.Fatalf("failed list disks: %s", err)
	}
	instances := listInstancesResponse.GetInstances()
	disks := listDisksResponse.GetDisks()

	r := &Resources{}
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "CPU", "RAM", "Network HDD", "Network SSD"})
	for _, instance := range instances {
		d := &Disks{}
		d.Add(instance, disks)
		t.AppendRow(r.GetRow(instance, d))
	}

	f := table.Row{
		"Total",
		fmt.Sprintf("%d", r.GetCores()),
		fmt.Sprintf("%.2f", r.GetMemory()),
		fmt.Sprintf("%.2f", r.Disks.GetNetworkHDD()),
		fmt.Sprintf("%.2f", r.Disks.GetNetworkSSD()),
	}
	t.AppendFooter(f)
	fmt.Println(t.Render())
}

func (d *Disks) Add(instance *compute.Instance, disks []*compute.Disk) {
	for _, disk := range disks {
		for _, i := range disk.GetInstanceIds() {
			if i == instance.GetId() {
				s := float64(disk.GetSize()) / MemCapacity

				switch disk.GetTypeId() {
				case "network-hdd":
					d.NetworkHDD += s
				case "network-ssd":
					d.NetworkSSD += s
				}
			}
		}
	}
}

func (d *Disks) GetNetworkHDD() float64 {
	return d.NetworkHDD
}

func (d *Disks) GetNetworkSSD() float64 {
	return d.NetworkSSD
}

func (r *Resources) GetRow(instance *compute.Instance, disks *Disks) table.Row {
	cores := instance.GetResources().GetCores()
	memory := float64(instance.GetResources().GetMemory()) / MemCapacity

	r.Add(cores, memory, disks)

	return table.Row{
		instance.GetName(),
		fmt.Sprintf("%d", cores),
		fmt.Sprintf("%.2f", memory),
		fmt.Sprintf("%.2f", disks.GetNetworkHDD()),
		fmt.Sprintf("%.2f", disks.GetNetworkSSD()),
	}
}

func (r *Resources) GetCores() int64 {
	return r.Cores
}

func (r *Resources) GetMemory() float64 {
	return r.Memory
}

func (r *Resources) Add(cores int64, memory float64, disks *Disks) {
	r.Cores += cores
	r.Memory += memory
	r.Disks.NetworkHDD += disks.GetNetworkHDD()
	r.Disks.NetworkSSD += disks.GetNetworkSSD()
}
