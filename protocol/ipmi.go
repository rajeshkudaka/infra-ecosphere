package protocol

import (
	"io"
	"encoding/binary"
	"bytes"
	"net"
	"log"
	"unsafe"
)

// port from OpenIPMI
// Network Functions
const (
	IPMI_NETFN_CHASSIS =		0x00
	IPMI_NETFN_BRIDGE =		0x02
	IPMI_NETFN_SENSOR_EVENT =	0x04
	IPMI_NETFN_APP =		0x06
	IPMI_NETFN_FIRMWARE =		0x08
	IPMI_NETFN_STORAGE  =		0x0a
	IPMI_NETFN_TRANSPORT =		0x0c
	IPMI_NETFN_GROUP_EXTENSION =	0x2c
	IPMI_NETFN_OEM_GROUP =		0x2e

	// Response Bit
	IPMI_NETFN_RESPONSE =		0x01
)

type IPMISessionWrapper struct {
	AuthenticationType uint8
	SequenceNumber uint32
	SessionId uint32
	MessageLen uint8
}

type IPMIMessage struct {
	TargetAddress uint8
	TargetLun uint8			// NetFn (6) + Lun (2)
	Checksum uint8
	SourceAddress uint8
	SourceLun uint8			// SequenceNumber (6) + Lun (2)
	Command uint8
	CompletionCode uint8
	Data []uint8
	DataChecksum uint8
}

// In Go 1.6, it doesn't support unalignment structure, so we use constant value here.
const (
	LEN_IPMISESSIONWRAPPER =	10
	LEN_IPMIMESSAGE_HEADER =	6
)

func DeserializeIPMI(buf io.Reader)  (length uint32, wrapper IPMISessionWrapper, message IPMIMessage) {
	length = 0

	binary.Read(buf, binary.BigEndian, &wrapper)
	length += LEN_IPMISESSIONWRAPPER

	binary.Read(buf, binary.BigEndian, &message.TargetAddress)
	binary.Read(buf, binary.BigEndian, &message.TargetLun)
	binary.Read(buf, binary.BigEndian, &message.Checksum)
	binary.Read(buf, binary.BigEndian, &message.SourceAddress)
	binary.Read(buf, binary.BigEndian, &message.SourceLun)
	binary.Read(buf, binary.BigEndian, &message.Command)
	length += LEN_IPMIMESSAGE_HEADER

	dataLen := wrapper.MessageLen - uint8(LEN_IPMIMESSAGE_HEADER) - 1
	if dataLen > 0 {
		message.Data = make([]uint8, dataLen, dataLen)
		binary.Read(buf, binary.BigEndian, &message.Data)
	}
	binary.Read(buf, binary.BigEndian, &message.DataChecksum)

	log.Println("    IPMI Session Wrapper Length = ", LEN_IPMISESSIONWRAPPER)
	log.Println("    IPMI Message Header Length = ", LEN_IPMIMESSAGE_HEADER)
	log.Println("    IPMI Message Data Length = ", dataLen)

	return length, wrapper, message
}

func SerializeIPMI(buf *bytes.Buffer, wrapper IPMISessionWrapper, message IPMIMessage) {
	// Calculate data checksum
	sum := uint32(0)
	sum += uint32(message.SourceAddress)
	sum += uint32(message.SourceLun)
	sum += uint32(message.Command)
	for i := 0; i < len(message.Data) ; i+=1 {
		sum += uint32(message.Data[i])
	}
	message.DataChecksum = uint8(0x100 - (sum & 0xff))

	// Calculate IPMI Message Checksum
	sum = uint32(message.TargetAddress) + uint32(message.TargetLun)
	message.Checksum = uint8(0x100 - (sum & 0xff))

	// Calculate Message Length
	length := uint32(0)
	length += uint32(unsafe.Sizeof(message.TargetAddress))
	length += uint32(unsafe.Sizeof(message.TargetLun))
	length += uint32(unsafe.Sizeof(message.Checksum))
	length += uint32(unsafe.Sizeof(message.SourceAddress))
	length += uint32(unsafe.Sizeof(message.SourceLun))
	length += uint32(unsafe.Sizeof(message.Command))
	length += uint32(unsafe.Sizeof(message.CompletionCode))
	length += uint32(len(message.Data))
	length += uint32(unsafe.Sizeof(message.DataChecksum))
	wrapper.MessageLen = uint8(length)

	// output
	binary.Write(buf, binary.BigEndian, wrapper)
	binary.Write(buf, binary.BigEndian, message.TargetAddress)
	binary.Write(buf, binary.BigEndian, message.TargetLun)
	binary.Write(buf, binary.BigEndian, message.Checksum)
	binary.Write(buf, binary.BigEndian, message.SourceAddress)
	binary.Write(buf, binary.BigEndian, message.SourceLun)
	binary.Write(buf, binary.BigEndian, message.Command)
	binary.Write(buf, binary.BigEndian, message.CompletionCode)
	buf.Write(message.Data)
	binary.Write(buf, binary.BigEndian, message.DataChecksum)
}

func BuildUpRMCPForIPMI() (rmcp RemoteManagementControlProtocol) {
	rmcp.Version = RMCP_VERSION_1
	rmcp.Reserved = 0x00
	rmcp.Sequence = 0xff
	rmcp.Class = RMCP_CLASS_IPMI

	return rmcp
}

func IPMIDeserializeAndExecute(buf io.Reader, addr *net.UDPAddr, server *net.UDPConn) {
	_, wrapper, message := DeserializeIPMI(buf)

	netFunction := (message.TargetLun & 0xFC) >> 2;

	switch netFunction {
	case IPMI_NETFN_CHASSIS:
		log.Println("    IPMI: NetFunction = CHASSIS")
	case IPMI_NETFN_BRIDGE:
		log.Println("    IPMI: NetFunction = BRIDGE")
	case IPMI_NETFN_SENSOR_EVENT:
		log.Println("    IPMI: NetFunction = SENSOR / EVENT")
	case IPMI_NETFN_APP:
		log.Println("    IPMI: NetFunction = APP")
		IPMI_APP_DeserializeAndExecute(addr, server, wrapper, message)
	case IPMI_NETFN_FIRMWARE:
		log.Println("    IPMI: NetFunction = FIRMWARE")
	case IPMI_NETFN_STORAGE:
		log.Println("    IPMI: NetFunction = STORAGE")
	case IPMI_NETFN_TRANSPORT:
		log.Println("    IPMI: NetFunction = TRANSPORT")
	case IPMI_NETFN_GROUP_EXTENSION:
		log.Println("    IPMI: NetFunction = GROUP EXTENSION")
	case IPMI_NETFN_OEM_GROUP:
		log.Println("    IPMI: NetFunction = OEM GROUP")
	}
}