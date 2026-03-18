package qris

import (
	"fmt"
	"strconv"
	"strings"
)

// calculateCRC16 calculates CRC-16 (CCITT-False)
func calculateCRC16(data string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// validateQRIS validates a static QRIS string
func validateQRIS(qrisString string) error {
	if len(qrisString) < 8 {
		return fmt.Errorf("qris string too short")
	}

	// 1. Must end with field "63" + 4-char CRC
	if !strings.HasSuffix(qrisString[:len(qrisString)-4], "6304") {
		return fmt.Errorf("qris string does not end with CRC field 6304")
	}

	// 2. Recalculate CRC on everything before the last 4 chars
	dataToCrc := qrisString[:len(qrisString)-4]
	expectedCrc := qrisString[len(qrisString)-4:]

	// 3. Compare — reject if mismatch
	calculatedCrc := calculateCRC16(dataToCrc)
	if calculatedCrc != expectedCrc {
		return fmt.Errorf("CRC mismatch: expected %s, got %s", expectedCrc, calculatedCrc)
	}

	// 4. Check Point of Initiation field "01" == "11" (static)
	if !strings.Contains(qrisString, "010211") {
		if strings.Contains(qrisString, "010212") {
			return fmt.Errorf("QRIS is already dynamic")
		}
		return fmt.Errorf("missing static point of initiation field (010211)")
	}

	return nil
}

// makeDynamic converts a static QRIS to dynamic by injecting the transaction amount
func makeDynamic(staticQRIS string, amount int64) (string, error) {
	// 1. Set field "01" value from "11" to "12"
	dynQRIS := strings.Replace(staticQRIS, "010211", "010212", 1)

	// 2. Inject field "54" (Transaction Amount) with amount string
	amountStr := strconv.FormatInt(amount, 10)
	amountLen := fmt.Sprintf("%02d", len(amountStr))
	field54 := "54" + amountLen + amountStr

	// Insert field 54 before field "58" (Country Code)
	idx58 := strings.Index(dynQRIS, "5802")
	if idx58 == -1 {
		return "", fmt.Errorf("country code field 58 not found")
	}

	dynQRISWithAmount := dynQRIS[:idx58] + field54 + dynQRIS[idx58:]

	// 3. Remove existing CRC field "63" value
	idx63 := strings.LastIndex(dynQRISWithAmount, "6304")
	if idx63 == -1 {
		return "", fmt.Errorf("CRC field 6304 not found")
	}
	dataToCrc := dynQRISWithAmount[:idx63+4]

	// 4. Recalculate CRC-16 on the new string + "6304"
	newCrc := calculateCRC16(dataToCrc)

	// 5. Append "6304" + new CRC
	finalQRIS := dataToCrc + newCrc

	return finalQRIS, nil
}
