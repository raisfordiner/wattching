package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/raisfordiner/wattching/pkg/cpuinfo"
	"github.com/raisfordiner/wattching/pkg/msr"
)

// unit_msr_t structure from the C code
type unitMSR struct {
	Power  uint8 // bits 0-3
	Energy uint8 // bits 8-12
	Time   uint8 // bits 16-19
}

type powerDomain struct {
	name         string
	msrOffset    int64
	lastEnergy   uint64
	hasDomain    bool
	currentPower float64
}

func parseUnitMSR(data uint64) unitMSR {
	return unitMSR{
		Power:  uint8(data & 0xF),
		Energy: uint8((data >> 8) & 0x1F),
		Time:   uint8((data >> 16) & 0xF),
	}
}

func main() {
	log.Println("Starting Wattching...")

	// Display CPU Info
	cpu := cpuinfo.GetCPUInfo()
	fmt.Println("--- CPU Information ---")
	fmt.Printf("Vendor: %s\n", cpu.VendorString)
	fmt.Printf("Model:  %s\n", cpu.BrandString)
	fmt.Println("-----------------------")

	msrFile, err := msr.OpenMSR(0)
	if err != nil {
		log.Fatalf("Error opening MSR on CPU 0: %v", err)
	}
	defer msrFile.Close()

	// Get unit info
	unitData, err := msrFile.ReadMSR(msr.UNIT_MULTIPLIER)
	if err != nil {
		log.Fatalf("Failed to read UNIT_MULTIPLIER MSR: %v", err)
	}
	units := parseUnitMSR(unitData)
	energyUnit := math.Pow(0.5, float64(units.Energy))
	log.Printf("Energy unit: 1/2^%d Joules (~%f J)", units.Energy, energyUnit)

	domains := []*powerDomain{
		{name: "Package", msrOffset: msr.PKG_STATUS},
		{name: "Cores (PP0)", msrOffset: msr.PP0_STATUS},
		{name: "GPU (PP1)", msrOffset: msr.PP1_STATUS},
		{name: "DRAM", msrOffset: msr.DRAM_STATUS},
	}

	for _, domain := range domains {
		has, err := msrFile.CheckMSR(domain.msrOffset)
		if err != nil {
			log.Printf("Warning: could not check for %s domain: %v", domain.name, err)
			continue
		}
		domain.hasDomain = has
		if has {
			initialEnergy, err := msrFile.ReadMSR(domain.msrOffset)
			if err != nil {
				log.Fatalf("Failed to read initial energy for %s: %v", domain.name, err)
			}
			domain.lastEnergy = initialEnergy
			log.Printf("Successfully initialized domain: %s", domain.name)
		} else {
			log.Printf("Domain not available: %s", domain.name)
		}
	}

	lastTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Println("\nWatching power consumption... Press Ctrl+C to stop.")

	for t := range ticker.C {
		timeElapsed := t.Sub(lastTime).Seconds()
		if timeElapsed == 0 {
			continue
		}

		for _, domain := range domains {
			if !domain.hasDomain {
				continue
			}

			currentEnergy, err := msrFile.ReadMSR(domain.msrOffset)
			if err != nil {
				log.Printf("Warning: Failed to read MSR for %s: %v", domain.name, err)
				continue
			}

			var energyConsumed uint64
			// Handle 32-bit counter wrap-around
			if currentEnergy < domain.lastEnergy {
				energyConsumed = (math.MaxUint32 - domain.lastEnergy) + currentEnergy
			} else {
				energyConsumed = currentEnergy - domain.lastEnergy
			}

			domain.currentPower = float64(energyConsumed) * energyUnit / timeElapsed
			domain.lastEnergy = currentEnergy
		}

		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println("--- CPU Information ---")
		fmt.Printf("Vendor: %s\n", cpu.VendorString)
		fmt.Printf("Model:  %s\n", cpu.BrandString)
		fmt.Println("-----------------------")
		fmt.Println("--- Current Power Consumption ---")
		for _, domain := range domains {
			if domain.hasDomain {
				fmt.Printf("% -15s: %.2f W\n", domain.name, domain.currentPower)
			}
		}
		fmt.Println("---------------------------------")

		lastTime = t
	}
}

