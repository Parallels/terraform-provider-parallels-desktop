package telemetry

import (
	"crypto/sha256"
	"encoding/base64"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	fingerprintOnce   sync.Once
	cachedFingerprint string
)

// MachineFingerprint returns a stable, anonymous unique identifier for the
// machine this provider process runs on. It is the SHA-256 (base64url,
// unpadded) of hostname + GOOS + GOARCH + the sorted set of non-loopback
// hardware (MAC) addresses.
//
// Every input is hashed; no raw identifier is ever transmitted. The value is
// computed once per process and cached. It is used as both the Amplitude
// user_id and device_id so that each machine counts as exactly one unique
// user, independent of how many people or licenses share it.
//
// Note: environments without stable MAC addresses (some containers/VMs) still
// get a deterministic ID from hostname+OS+arch, but it is less discriminating.
func MachineFingerprint() string {
	fingerprintOnce.Do(func() {
		hostname, _ := os.Hostname()

		h := sha256.New()
		for _, part := range []string{hostname, runtime.GOOS, runtime.GOARCH, macAddressSet()} {
			h.Write([]byte(part))
			h.Write([]byte{0}) // domain separator so inputs cannot be confused
		}
		cachedFingerprint = base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	})

	return cachedFingerprint
}

// macAddressSet returns the sorted, de-duplicated set of MAC addresses of all
// non-loopback interfaces, joined with ",". Sorting and ignoring the
// up/down state keep the value stable against interface enumeration order
// and link-state changes.
func macAddressSet() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	seen := make(map[string]struct{}, len(interfaces))
	macs := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}
		if _, dup := seen[mac]; dup {
			continue
		}
		seen[mac] = struct{}{}
		macs = append(macs, mac)
	}

	sort.Strings(macs)
	return strings.Join(macs, ",")
}
