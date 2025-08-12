package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/raisfordiner/wattching/pkg/cpuinfo"
	"github.com/raisfordiner/wattching/pkg/msr"
	"github.com/raisfordiner/wattching/pkg/tui"
)

type unitMSR struct {
	Energy uint8 // bits 8-12
}

type powerDomain struct {
	name       string
	msrOffset  int64
	lastEnergy uint64
	hasDomain  bool
}

func parseUnitMSR(data uint64) unitMSR {
	return unitMSR{
		Energy: uint8((data >> 8) & 0x1F),
	}
}

func main() {
	cpuInfo := cpuinfo.GetCPUInfo()

	msrFile, err := msr.OpenMSR(0)
	if err != nil {
		log.Fatalf("Error opening MSR on CPU 0: %v", err)
	}
	defer msrFile.Close()

	unitData, err := msrFile.ReadMSR(msr.UNIT_MULTIPLIER)
	if err != nil {
		log.Fatalf("Failed to read UNIT_MULTIPLIER MSR: %v", err)
	}
	units := parseUnitMSR(unitData)
	energyUnit := math.Pow(0.5, float64(units.Energy))

	domains := []*powerDomain{
		{name: "Package", msrOffset: msr.PKG_STATUS},
		{name: "Cores (PP0)", msrOffset: msr.PP0_STATUS},
		{name: "GPU (PP1)", msrOffset: msr.PP1_STATUS},
		{name: "DRAM", msrOffset: msr.DRAM_STATUS},
	}

	var orderedDomains []string
	for _, domain := range domains {
		has, err := msrFile.CheckMSR(domain.msrOffset)
		if err != nil {
			log.Fatalf("Could not check for %s domain: %v", domain.name, err)
		}
		domain.hasDomain = has
		if has {
			initialEnergy, err := msrFile.ReadMSR(domain.msrOffset)
			if err != nil {
				log.Fatalf("Failed to read initial energy for %s: %v", domain.name, err)
			}
			domain.lastEnergy = initialEnergy
			orderedDomains = append(orderedDomains, domain.name)
		}
	}

	p := tea.NewProgram(tui.InitialModel(cpuInfo, orderedDomains))

	// Power Monitoring Goroutine
	go func() {
		lastTime := time.Now()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			currentTime := time.Now()
			timeElapsed := currentTime.Sub(lastTime).Seconds()
			if timeElapsed == 0 {
				continue
			}

			powerUpdate := make(map[string]float64)

			for _, domain := range domains {
				if !domain.hasDomain {
					continue
				}

				currentEnergy, err := msrFile.ReadMSR(domain.msrOffset)
				if err != nil {
					p.Send(tui.ErrorMsg{Err: fmt.Errorf("failed to read MSR for %s: %w", domain.name, err)})
					return
				}

				var energyConsumed uint64
				if currentEnergy < domain.lastEnergy {
					energyConsumed = (math.MaxUint32 - domain.lastEnergy) + currentEnergy
				} else {
					energyConsumed = currentEnergy - domain.lastEnergy
				}

				powerUpdate[domain.name] = float64(energyConsumed) * energyUnit / timeElapsed
				domain.lastEnergy = currentEnergy
			}

			p.Send(tui.PowerUpdateMsg{Data: powerUpdate})
			lastTime = currentTime
		}
	}()

	// Run TUI
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
