package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-sdk"
)

const (
	MemCapacity = 1024 * 1024 * 1024
)

type Resources struct {
	PlatformId string
	Cores      int64
	Memory     float64
	Disks
}

type Disks struct {
	NetworkHDD float64
	NetworkSSD float64
}

func main() {
	folderID := flag.String("folder-id", "", "specify folder id")
	token := flag.String("token", "", "specify token")
	instanceNamePrefix := flag.String("instance-name-prefix", "", "specify instances name prefix")
	outputFormat := flag.String("output-format", "table", "specify result output format")
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

	switch *outputFormat {
	case "csv":
		printCsvOutput(instances, disks, *instanceNamePrefix)
	default:
		printTableOutput(instances, disks, *instanceNamePrefix)
	}
}

func printCsvOutput(instances []*compute.Instance, disks []*compute.Disk, prefix string) {
	for _, instance := range instances {
		if prefix != "" {
			if !strings.HasPrefix(instance.GetName(), prefix) {
				continue
			}
		}

		d := &Disks{}
		d.Add(instance, disks)

		fmt.Printf(
			"%s,%s,%d,%.2f,%.2f,%.2f\n",
			instance.GetName(),
			instance.GetPlatformId(),
			instance.GetResources().GetCores(),
			float64(instance.GetResources().GetMemory())/MemCapacity,
			d.GetNetworkHDD(),
			d.GetNetworkSSD(),
		)
	}
}

func printTableOutput(instances []*compute.Instance, disks []*compute.Disk, prefix string) {
	r := &Resources{}
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "Platform", "CPU", "RAM", "Network HDD", "Network SSD"})
	for _, instance := range instances {
		if prefix != "" {
			if !strings.HasPrefix(instance.GetName(), prefix) {
				continue
			}
		}

		d := &Disks{}
		d.Add(instance, disks)
		t.AppendRow(r.GetRow(instance, d))
	}

	f := table.Row{
		"Total",
		fmt.Sprintf("%s", ""),
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
	platform := instance.GetPlatformId()
	cores := instance.GetResources().GetCores()
	memory := float64(instance.GetResources().GetMemory()) / MemCapacity

	r.Add(cores, memory, disks)

	return table.Row{
		instance.GetName(),
		fmt.Sprintf("%s", platform),
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
