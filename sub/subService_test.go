package sub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestRewriteLinkUri(t *testing.T) {
	// 1. Test VMess rewrite
	vmessObj := map[string]interface{}{
		"v":    "2",
		"ps":   "test-vmess",
		"add":  "old.host.com",
		"port": float64(12345),
		"id":   "uuid-placeholder",
	}
	vmessBytes, _ := json.Marshal(vmessObj)
	vmessUri := "vmess://" + base64.StdEncoding.EncodeToString(vmessBytes)

	rewrittenVMess := rewriteLinkUri(vmessUri, "new.host.com", 54321)
	if !strings.HasPrefix(rewrittenVMess, "vmess://") {
		t.Fatalf("Expected vmess:// prefix, got %s", rewrittenVMess)
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(rewrittenVMess, "vmess://"))
	if err != nil {
		t.Fatalf("Failed to decode rewritten vmess base64: %v", err)
	}
	var newVmessObj map[string]interface{}
	json.Unmarshal(decodedBytes, &newVmessObj)

	if newVmessObj["add"] != "new.host.com" {
		t.Errorf("Expected add to be new.host.com, got %v", newVmessObj["add"])
	}
	if newVmessObj["port"] != float64(54321) {
		t.Errorf("Expected port to be 54321, got %v", newVmessObj["port"])
	}

	// 2. Test VMess rewrite without port mapping (targetPort = 0)
	rewrittenVMessNoPort := rewriteLinkUri(vmessUri, "new.host.com", 0)
	decodedBytes2, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(rewrittenVMessNoPort, "vmess://"))
	var newVmessObj2 map[string]interface{}
	json.Unmarshal(decodedBytes2, &newVmessObj2)
	if newVmessObj2["port"] != float64(12345) {
		t.Errorf("Expected port to remain 12345, got %v", newVmessObj2["port"])
	}

	// 3. Test VLESS rewrite
	vlessUri := "vless://uuid-placeholder@old.host.com:12345?security=none#test-vless"
	rewrittenVless := rewriteLinkUri(vlessUri, "new.host.com", 54321)
	expectedVless := "vless://uuid-placeholder@new.host.com:54321?security=none#test-vless"
	if rewrittenVless != expectedVless {
		t.Errorf("VLESS rewrite failed: expected %s, got %s", expectedVless, rewrittenVless)
	}

	// 4. Test Trojan rewrite
	trojanUri := "trojan://password-placeholder@old.host.com:12345?security=tls#test-trojan"
	rewrittenTrojan := rewriteLinkUri(trojanUri, "new.host.com", 0)
	expectedTrojan := "trojan://password-placeholder@new.host.com:12345?security=tls#test-trojan"
	if rewrittenTrojan != expectedTrojan {
		t.Errorf("Trojan rewrite failed: expected %s, got %s", expectedTrojan, rewrittenTrojan)
	}

	// 5. Test ShadowSocks rewrite
	ssUri := "ss://method-pass@old.host.com:12345#test-ss"
	rewrittenSS := rewriteLinkUri(ssUri, "new.host.com", 54321)
	expectedSS := "ss://method-pass@new.host.com:54321#test-ss"
	if rewrittenSS != expectedSS {
		t.Errorf("SS rewrite failed: expected %s, got %s", expectedSS, rewrittenSS)
	}
}
