package spp

const (
	CmdGetProtocolVersion    = 0xC001
	CmdGetStatus             = 0xC00A
	CmdGetFirmwareVersion    = 0xC042
	CmdGetBattery            = 0xC007
	CmdGetIdentity           = 0xC005
	CmdGetRemoteConfig       = 0xC006
	CmdGetSupportedFeature   = 0xC00D
	CmdGetEQMode             = 0xC01F
	CmdGetNoiseReduction     = 0xC01E
	CmdGetSpatialAudio       = 0xC04F
	CmdGetLagMode            = 0xC041
	CmdGetDualEnable         = 0xC027
	CmdGetDualDeviceList     = 0xC028
	CmdSetEQMode             = 0xF010
	CmdSetNoiseReduction     = 0xF00F
	CmdSetSpatialAudio       = 0xF052
	CmdSetLagMode            = 0xF040
	CmdSetDualEnable         = 0xF01A
	CmdSetConnectDevice      = 0xF01B
	CmdAckSetLagMode         = 0x7040
	CmdAckSetNoiseReduction  = 0x700F
	CmdAckSetEQMode          = 0x7010
	CmdAckSetDualEnable      = 0x701A
	CmdAckSetConnectDevice   = 0x701B
	CmdAckSetSpatialAudio    = 0x7052
	CmdBatteryChanged        = 0xE001
	CmdNoiseReductionChanged = 0xE003
	CmdDualSwitchChanged     = 0xE006
	CmdBattery               = 0xE007
	CmdBudsBattery           = 0xE005
	CmdStatus                = 0xE002
	CmdIdentity              = 0xE008
	CmdDualConnectChanged    = 0xE00E
	CmdLagModeChanged        = 0xE019
)

