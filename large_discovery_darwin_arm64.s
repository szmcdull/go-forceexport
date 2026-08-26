//go:build go1.23 && darwin && large_discovery_test
// +build go1.23,darwin,large_discovery_test

#include "textflag.h"

GLOBL ·largeDiscoveryPadding(SB), RODATA|NOPTR, $41943040

TEXT ·touchLargeDiscoveryPadding(SB), NOSPLIT, $0-0
	MOVD $·largeDiscoveryPadding(SB), R0
	RET
