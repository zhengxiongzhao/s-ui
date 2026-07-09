package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util/common"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WarpService struct{}

func (s *WarpService) getWarpInfo(deviceId string, accessToken string) ([]byte, error) {
	url := fmt.Sprintf("https://api.cloudflareclient.com/v0a2158/reg/%s", deviceId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, common.NewErrorf("warp: unexpected status code %d", resp.StatusCode)
	}
	buffer := bytes.NewBuffer(make([]byte, 8192))
	buffer.Reset()
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *WarpService) RegisterWarp(ep *model.Endpoint) error {
	tos := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	privateKey, _ := wgtypes.GenerateKey()
	publicKey := privateKey.PublicKey().String()
	hostName, _ := os.Hostname()

	data := fmt.Sprintf(`{"key":"%s","tos":"%s","type": "PC","model": "s-ui", "name": "%s"}`, publicKey, tos, hostName)
	url := "https://api.cloudflareclient.com/v0a2158/reg"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return err
	}

	req.Header.Add("CF-Client-Version", "a-7.21-0721")
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return common.NewErrorf("warp: registration failed with status code %d", resp.StatusCode)
	}
	buffer := bytes.NewBuffer(make([]byte, 8192))
	buffer.Reset()
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	var rspData map[string]interface{}
	err = json.Unmarshal(buffer.Bytes(), &rspData)
	if err != nil {
		return err
	}

	deviceId, ok := rspData["id"].(string)
	if !ok || deviceId == "" {
		return errors.New("warp register response missing device id")
	}
	token, ok := rspData["token"].(string)
	if !ok || token == "" {
		return errors.New("warp register response missing token")
	}
	account, ok := rspData["account"].(map[string]interface{})
	if !ok {
		logger.Debug("Error accessing account value.")
		return errors.New("warp register response missing account")
	}
	license, ok := account["license"].(string)
	if !ok {
		logger.Debug("Error accessing license value.")
		return errors.New("warp register response missing license")
	}

	warpInfo, err := s.getWarpInfo(deviceId, token)
	if err != nil {
		return err
	}

	var warpDetails map[string]interface{}
	err = json.Unmarshal(warpInfo, &warpDetails)
	if err != nil {
		return err
	}

	warpConfig, ok := warpDetails["config"].(map[string]interface{})
	if !ok {
		return errors.New("warp details response missing config")
	}
	clientId, _ := warpConfig["client_id"].(string)
	reserved := s.getReserved(clientId)
	interfaceConfig, ok := warpConfig["interface"].(map[string]interface{})
	if !ok {
		return errors.New("warp details response missing interface")
	}
	addresses, ok := interfaceConfig["addresses"].(map[string]interface{})
	if !ok {
		return errors.New("warp details response missing addresses")
	}
	v4, _ := addresses["v4"].(string)
	v6, _ := addresses["v6"].(string)
	peersData, ok := warpConfig["peers"].([]interface{})
	if !ok || len(peersData) == 0 {
		return errors.New("warp details response missing peers")
	}
	peer, ok := peersData[0].(map[string]interface{})
	if !ok {
		return errors.New("warp details response has invalid peer")
	}
	endpoint, ok := peer["endpoint"].(map[string]interface{})
	if !ok {
		return errors.New("warp details response missing peer endpoint")
	}
	peerEndpoint, ok := endpoint["host"].(string)
	if !ok || peerEndpoint == "" {
		return errors.New("warp details response missing peer endpoint host")
	}
	peerEpAddress, peerEpPort, err := net.SplitHostPort(peerEndpoint)
	if err != nil {
		return err
	}
	peerPublicKey, _ := peer["public_key"].(string)
	peerPort, _ := strconv.Atoi(peerEpPort)

	peers := []map[string]interface{}{
		{
			"address":     peerEpAddress,
			"port":        peerPort,
			"public_key":  peerPublicKey,
			"allowed_ips": []string{"0.0.0.0/0", "::/0"},
			"reserved":    reserved,
		},
	}

	warpData := map[string]interface{}{
		"access_token": token,
		"device_id":    deviceId,
		"license_key":  license,
	}

	ep.Ext, err = json.MarshalIndent(warpData, "", "  ")
	if err != nil {
		return err
	}

	var epOptions map[string]interface{}
	err = json.Unmarshal(ep.Options, &epOptions)
	if err != nil {
		return err
	}
	if epOptions == nil {
		epOptions = make(map[string]interface{})
	}
	epOptions["private_key"] = privateKey.String()
	epOptions["address"] = []string{fmt.Sprintf("%s/32", v4), fmt.Sprintf("%s/128", v6)}
	epOptions["listen_port"] = 0
	epOptions["peers"] = peers

	ep.Options, err = json.MarshalIndent(epOptions, "", "  ")
	return err
}

func (s *WarpService) getReserved(clientID string) []int {
	var reserved []int
	decoded, err := base64.StdEncoding.DecodeString(clientID)
	if err != nil {
		return nil
	}

	hexString := ""
	for _, char := range decoded {
		hex := fmt.Sprintf("%02x", char)
		hexString += hex
	}

	for i := 0; i < len(hexString); i += 2 {
		hexByte := hexString[i : i+2]
		decValue, err := strconv.ParseInt(hexByte, 16, 32)
		if err != nil {
			return nil
		}
		reserved = append(reserved, int(decValue))
	}

	return reserved
}

func (s *WarpService) SetWarpLicense(old_license string, ep *model.Endpoint) error {
	var warpData map[string]string
	err := json.Unmarshal(ep.Ext, &warpData)
	if err != nil {
		return err
	}

	if warpData["license_key"] == old_license {
		return nil
	}

	url := fmt.Sprintf("https://api.cloudflareclient.com/v0a2158/reg/%s/account", warpData["device_id"])
	data := fmt.Sprintf(`{"license": "%s"}`, warpData["license_key"])

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+warpData["access_token"])

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buffer := bytes.NewBuffer(make([]byte, 8192))
	buffer.Reset()
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	var response map[string]interface{}
	err = json.Unmarshal(buffer.Bytes(), &response)
	if err != nil {
		return err
	}

	if success, ok := response["success"].(bool); ok && success == false {
		errorArr, _ := response["errors"].([]interface{})
		if len(errorArr) == 0 {
			return errors.New("warp license update failed")
		}
		errorObj, _ := errorArr[0].(map[string]interface{})
		if errorObj == nil {
			return errors.New("warp license update failed with invalid error response")
		}
		return common.NewError(errorObj["code"], errorObj["message"])
	}

	return nil
}
