package apimodels

type VirtualMachine struct {
	User                  string                             `json:"user"`
	ID                    string                             `json:"ID"`
	HostId                string                             `json:"host_id"`
	HostExternalIpAddress string                             `json:"host_external_ip_address"`
	InternalIpAddress     string                             `json:"internal_ip_address"`
	Name                  string                             `json:"Name"`
	Description           string                             `json:"Description"`
	Type                  string                             `json:"Type"`
	State                 string                             `json:"State"`
	OS                    string                             `json:"OS"`
	Template              string                             `json:"Template"`
	Uptime                string                             `json:"Uptime"`
	HomePath              string                             `json:"Home path"`
	Home                  string                             `json:"Home"`
	RestoreImage          string                             `json:"Restore Image"`
	GuestTools            VirtualMachineGuestTools           `json:"GuestTools"`
	MouseAndKeyboard      VirtualMachineMouseAndKeyboard     `json:"Mouse and Keyboard"`
	USBAndBluetooth       VirtualMachineUSBAndBluetooth      `json:"USB and Bluetooth"`
	StartupAndShutdown    VirtualMachineStartupAndShutdown   `json:"Startup and Shutdown"`
	Optimization          VirtualMachineOptimization         `json:"Optimization"`
	TravelMode            VirtualMachineTravelMode           `json:"Travel mode"`
	Security              VirtualMachineSecurity             `json:"Security"`
	SmartGuard            VirtualMachineExpiration           `json:"Smart Guard"`
	Modality              VirtualMachineModality             `json:"Modality"`
	Fullscreen            VirtualMachineFullscreen           `json:"Fullscreen"`
	Coherence             VirtualMachineCoherence            `json:"Coherence"`
	TimeSynchronization   VirtualMachineTimeSynchronization  `json:"Time Synchronization"`
	Expiration            VirtualMachineExpiration           `json:"Expiration"`
	BootOrder             string                             `json:"Boot order"`
	BIOSType              string                             `json:"BIOS type"`
	EFISecureBoot         string                             `json:"EFI Secure boot"`
	AllowSelectBootDevice string                             `json:"Allow select boot device"`
	ExternalBootDevice    string                             `json:"External boot device"`
	SMBIOSSettings        VirtualMachineSMBIOSSettings       `json:"SMBIOS settings"`
	Hardware              VirtualMachineHardware             `json:"Hardware"`
	HostSharedFolders     VirtualMachineExpiration           `json:"Host Shared Folders"`
	HostDefinedSharing    string                             `json:"Host defined sharing"`
	SharedProfile         VirtualMachineExpiration           `json:"Shared Profile"`
	SharedApplications    VirtualMachineSharedApplications   `json:"Shared Applications"`
	SmartMount            VirtualMachineExpiration           `json:"SmartMount"`
	MiscellaneousSharing  VirtualMachineMiscellaneousSharing `json:"Miscellaneous Sharing"`
	Advanced              VirtualMachineAdvanced             `json:"Advanced"`
}

type VirtualMachineAdvanced struct {
	VMHostnameSynchronization    string `json:"VM hostname synchronization"`
	PublicSSHKeysSynchronization string `json:"Public SSH keys synchronization"`
	ShowDeveloperTools           string `json:"Show developer tools"`
	SwipeFromEdges               string `json:"Swipe from edges"`
	ShareHostLocation            string `json:"Share host location"`
}

type VirtualMachineCoherence struct {
	ShowWindowsSystrayInMACMenu string `json:"Show Windows systray in Mac menu"`
	AutoSwitchToFullScreen      string `json:"Auto-switch to full screen"`
	DisableAero                 string `json:"Disable aero"`
	HideMinimizedWindows        string `json:"Hide minimized windows"`
}

type VirtualMachineExpiration struct {
	Enabled bool `json:"enabled"`
}

type VirtualMachineFullscreen struct {
	UseAllDisplays        string `json:"Use all displays"`
	ActivateSpacesOnClick string `json:"Activate spaces on click"`
	OptimizeForGames      string `json:"Optimize for games"`
	GammaControl          string `json:"Gamma control"`
	ScaleViewMode         string `json:"Scale view mode"`
}

type VirtualMachineGuestTools struct {
	State   string `json:"state"`
	Version string `json:"version"`
}

type VirtualMachineHardware struct {
	CPU         VirtualMachineCPU         `json:"cpu"`
	Memory      VirtualMachineMemory      `json:"memory"`
	Video       VirtualMachineVideo       `json:"video"`
	MemoryQuota VirtualMachineMemoryQuota `json:"memory_quota"`
	Hdd0        VirtualMachineHdd0        `json:"hdd0"`
	Cdrom0      VirtualMachineCdrom0      `json:"cdrom0"`
	USB         VirtualMachineExpiration  `json:"usb"`
	Net0        VirtualMachineNet0        `json:"net0"`
	Sound0      VirtualMachineSound0      `json:"sound0"`
}

type VirtualMachineCPU struct {
	Cpus    int64  `json:"cpus"`
	Auto    string `json:"auto"`
	VTX     bool   `json:"VT-x"`
	Hotplug bool   `json:"hotplug"`
	Accl    string `json:"accl"`
	Mode    string `json:"mode"`
	Type    string `json:"type"`
}

type VirtualMachineCdrom0 struct {
	Enabled bool   `json:"enabled"`
	Port    string `json:"port"`
	Image   string `json:"image"`
	State   string `json:"state"`
}

type VirtualMachineHdd0 struct {
	Enabled       bool   `json:"enabled"`
	Port          string `json:"port"`
	Image         string `json:"image"`
	Type          string `json:"type"`
	Size          string `json:"size"`
	OnlineCompact string `json:"online-compact"`
}

type VirtualMachineMemory struct {
	Size    string `json:"size"`
	Auto    string `json:"auto"`
	Hotplug bool   `json:"hotplug"`
}

type VirtualMachineMemoryQuota struct {
	Auto string `json:"auto"`
}

type VirtualMachineNet0 struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	MAC     string `json:"mac"`
	Card    string `json:"card"`
}

type VirtualMachineSound0 struct {
	Enabled bool   `json:"enabled"`
	Output  string `json:"output"`
	Mixer   string `json:"mixer"`
}

type VirtualMachineVideo struct {
	AdapterType           string `json:"adapter-type"`
	Size                  string `json:"size"`
	The3DAcceleration     string `json:"3d-acceleration"`
	VerticalSync          string `json:"vertical-sync"`
	HighResolution        string `json:"high-resolution"`
	HighResolutionInGuest string `json:"high-resolution-in-guest"`
	NativeScalingInGuest  string `json:"native-scaling-in-guest"`
	AutomaticVideoMemory  string `json:"automatic-video-memory"`
}

type VirtualMachineMiscellaneousSharing struct {
	SharedClipboard string `json:"Shared clipboard"`
	SharedCloud     string `json:"Shared cloud"`
}

type VirtualMachineModality struct {
	OpacityPercentage  int64  `json:"Opacity (percentage)"`
	StayOnTop          string `json:"Stay on top"`
	ShowOnAllSpaces    string `json:"Show on all spaces "`
	CaptureMouseClicks string `json:"Capture mouse clicks"`
}

type VirtualMachineMouseAndKeyboard struct {
	SmartMouseOptimizedForGames string `json:"Smart mouse optimized for games"`
	StickyMouse                 string `json:"Sticky mouse"`
	SmoothScrolling             string `json:"Smooth scrolling"`
	KeyboardOptimizationMode    string `json:"Keyboard optimization mode"`
}

type VirtualMachineOptimization struct {
	FasterVirtualMachine     string `json:"Faster virtual machine"`
	HypervisorType           string `json:"Hypervisor type"`
	AdaptiveHypervisor       string `json:"Adaptive hypervisor"`
	DisabledWindowsLogo      string `json:"Disabled Windows logo"`
	AutoCompressVirtualDisks string `json:"Auto compress virtual disks"`
	NestedVirtualization     string `json:"Nested virtualization"`
	PMUVirtualization        string `json:"PMU virtualization"`
	LongerBatteryLife        string `json:"Longer battery life"`
	ShowBatteryStatus        string `json:"Show battery status"`
	ResourceQuota            string `json:"Resource quota"`
}

type VirtualMachineSMBIOSSettings struct {
	BIOSVersion        string `json:"BIOS Version"`
	SystemSerialNumber string `json:"System serial number"`
	BoardManufacturer  string `json:"Board Manufacturer"`
}

type VirtualMachineSecurity struct {
	Encrypted                string `json:"Encrypted"`
	TPMEnabled               string `json:"TPM enabled"`
	TPMType                  string `json:"TPM type"`
	CustomPasswordProtection string `json:"Custom password protection"`
	ConfigurationIsLocked    string `json:"Configuration is locked"`
	Protected                string `json:"Protected"`
	Archived                 string `json:"Archived"`
	Packed                   string `json:"Packed"`
}

type VirtualMachineSharedApplications struct {
	Enabled                      bool   `json:"enabled"`
	HostToGuestAppsSharing       string `json:"Host-to-guest apps sharing"`
	GuestToHostAppsSharing       string `json:"Guest-to-host apps sharing"`
	ShowGuestAppsFolderInDock    string `json:"Show guest apps folder in Dock"`
	ShowGuestNotifications       string `json:"Show guest notifications"`
	BounceDockIconWhenAppFlashes string `json:"Bounce dock icon when app flashes"`
}

type VirtualMachineStartupAndShutdown struct {
	Autostart      string `json:"Autostart"`
	AutostartDelay int64  `json:"Autostart delay"`
	Autostop       string `json:"Autostop"`
	StartupView    string `json:"Startup view"`
	OnShutdown     string `json:"On shutdown"`
	OnWindowClose  string `json:"On window close"`
	PauseIdle      string `json:"Pause idle"`
	UndoDisks      string `json:"Undo disks"`
}

type VirtualMachineTimeSynchronization struct {
	Enabled                         bool   `json:"enabled"`
	SmartMode                       string `json:"Smart mode"`
	IntervalInSeconds               int64  `json:"Interval (in seconds)"`
	TimezoneSynchronizationDisabled string `json:"Timezone synchronization disabled"`
}

type VirtualMachineTravelMode struct {
	EnterCondition string `json:"Enter condition"`
	EnterThreshold int64  `json:"Enter threshold"`
	QuitCondition  string `json:"Quit condition"`
}

type VirtualMachineUSBAndBluetooth struct {
	AutomaticSharingCameras    string `json:"Automatic sharing cameras"`
	AutomaticSharingBluetooth  string `json:"Automatic sharing bluetooth"`
	AutomaticSharingSmartCards string `json:"Automatic sharing smart cards"`
	AutomaticSharingGamepads   string `json:"Automatic sharing gamepads"`
	SupportUSB30               string `json:"Support USB 3.0"`
}
