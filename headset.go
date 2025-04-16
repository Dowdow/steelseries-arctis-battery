package main

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	VENDOR_ID  = 0x1038 // SteelSeries VID
	PRODUCT_ID = 0x12AD // Arctis 7 PID
)

// Constantes SetupAPI
const (
	DIGCF_PRESENT         = 0x00000002
	DIGCF_DEVICEINTERFACE = 0x00000010
)

// Structure pour stocker les attributs HID
type HIDD_ATTRIBUTES struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type SP_DEVICE_INTERFACE_DATA struct {
	cbSize             uint32
	InterfaceClassGuid windows.GUID
	Flags              uint32
	Reserved           uintptr
}

type SP_DEVICE_INTERFACE_DETAIL_DATA_W struct {
	cbSize     uint32
	DevicePath [1]uint16
}

type HIDP_CAPS struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

// GUID pour les périphériques HID
var GUID_DEVINTERFACE_HID = windows.GUID{
	Data1: 0x4d1e55b2,
	Data2: 0xf16f,
	Data3: 0x11cf,
	Data4: [8]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
}

// Charger les DLLs nécessaires
var (
	hid      = windows.NewLazySystemDLL("hid.dll")
	setupapi = windows.NewLazySystemDLL("setupapi.dll")

	procHidD_GetAttributes               = hid.NewProc("HidD_GetAttributes")
	procHidD_GetPreparsedData            = hid.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData           = hid.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps                     = hid.NewProc("HidP_GetCaps")
	procSetupDiGetClassDevsW             = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

func getBatteryLevel() (int, error) {
	// Obtenir la liste des périphériques HID
	deviceInfoSet, _, err := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&GUID_DEVINTERFACE_HID)),
		0,
		0,
		uintptr(DIGCF_PRESENT|DIGCF_DEVICEINTERFACE),
	)

	if deviceInfoSet == 0 {
		return -1, fmt.Errorf("SetupDiGetClassDevs a échoué: %v", err)
	}
	defer procSetupDiDestroyDeviceInfoList.Call(deviceInfoSet)

	// Parcourir chaque interface
	var deviceInterfaceData SP_DEVICE_INTERFACE_DATA
	deviceInterfaceData.cbSize = uint32(unsafe.Sizeof(deviceInterfaceData))

	for memberIndex := uint32(0); ; memberIndex++ {
		// Énumérer les interfaces de périphériques
		ret, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			deviceInfoSet,
			0,
			uintptr(unsafe.Pointer(&GUID_DEVINTERFACE_HID)),
			uintptr(memberIndex),
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
		)

		if ret == 0 {
			break // Plus d'interfaces
		}

		// Obtenir la taille requise pour les détails
		var deviceInterfaceDetailDataSize uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(
			deviceInfoSet,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			0,
			0,
			uintptr(unsafe.Pointer(&deviceInterfaceDetailDataSize)),
			0,
		)

		// Allouer la mémoire pour les détails
		buf := make([]byte, deviceInterfaceDetailDataSize)
		detailData := (*SP_DEVICE_INTERFACE_DETAIL_DATA_W)(unsafe.Pointer(&buf[0]))

		// Sous Windows 64-bit, cbSize doit être 8, sous Windows 32-bit, cbSize doit être 6
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detailData.cbSize = 8
		} else {
			detailData.cbSize = 6
		}

		// Obtenir les détails de l'interface
		ret, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(
			deviceInfoSet,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			uintptr(unsafe.Pointer(detailData)),
			uintptr(deviceInterfaceDetailDataSize),
			0,
			0,
		)

		if ret == 0 {
			continue
		}

		// Convertir le chemin du périphérique en string
		path := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&detailData.DevicePath[0])))

		// Ouvrir le périphérique
		handle, err := windows.CreateFile(
			windows.StringToUTF16Ptr(path),
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)

		if err != nil {
			continue
		}

		// Vérifier si c'est notre périphérique
		var attrs HIDD_ATTRIBUTES
		attrs.Size = uint32(unsafe.Sizeof(attrs))

		ret, _, _ = procHidD_GetAttributes.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&attrs)),
		)

		if ret == 0 {
			windows.CloseHandle(handle)
			continue
		}

		// Vérifier si c'est notre périphérique cible
		if attrs.VendorID != VENDOR_ID || attrs.ProductID != PRODUCT_ID {
			windows.CloseHandle(handle)
			continue
		}

		fmt.Printf("Périphérique trouvé - VID: 0x%04X, PID: 0x%04X\n", attrs.VendorID, attrs.ProductID)
		fmt.Printf("Chemin: %s\n", path)

		// Obtenir les capacités du périphérique
		var preparsedData uintptr
		ret, _, _ = procHidD_GetPreparsedData.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&preparsedData)),
		)
		if ret == 0 {
			fmt.Printf("HidD_GetPreparsedData a échoué\n")
			continue
		}
		defer procHidD_FreePreparsedData.Call(preparsedData)

		var caps HIDP_CAPS
		ret, _, _ = procHidP_GetCaps.Call(
			preparsedData,
			uintptr(unsafe.Pointer(&caps)),
		)
		if ret == 0 {
			fmt.Printf("HidP_GetCaps a échoué\n")
			continue
		}

		fmt.Printf("Taille du rapport d'entrée: %d octets\n", caps.InputReportByteLength)
		fmt.Printf("Taille du rapport de sortie: %d octets\n", caps.OutputReportByteLength)
		fmt.Printf("Taille du rapport de caractéristique: %d octets\n", caps.FeatureReportByteLength)

		// Créer un buffer de la taille appropriée
		outputBuffer := make([]byte, caps.OutputReportByteLength)
		// Copier la commande dans le buffer
		copy(outputBuffer, []byte{0x06, 0x18})

		// Envoyer la commande
		var bytesWritten uint32
		err = windows.WriteFile(handle, outputBuffer, &bytesWritten, nil)
		if err != nil {
			fmt.Printf("Échec de l'envoi de la commande (WriteFile): %v\n", err)
			continue
		}

		// Attendre un peu pour laisser le périphérique traiter la commande
		time.Sleep(20 * time.Millisecond)

		// Lire la réponse
		inputBuffer := make([]byte, caps.InputReportByteLength)
		var bytesRead uint32
		err = windows.ReadFile(handle, inputBuffer, &bytesRead, nil)
		if err != nil {
			fmt.Printf("Échec de la lecture de la réponse (ReadFile): %v\n", err)
			continue
		}

		return int(inputBuffer[2]), nil
	}

	return -1, fmt.Errorf("périphérique non trouvé ou niveau de batterie non disponible")
}
