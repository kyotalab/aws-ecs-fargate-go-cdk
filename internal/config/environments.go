package config

import (
	"fmt"
)

// EnvironmentConfig 環境別設定
type EnvironmentConfig struct {
	Name              string
	VpcCidr           string
	MaxAzs            int
	EnableNATGateway  bool
	EnableVPCFlowLogs bool

	// セキュリティ設定
	AllowSSHAccess  bool
	RestrictedCIDRs []string

	// タグ設定
	Tags map[string]string
}

// NetworkConfig ネットワーク固有の設定
type NetworkConfig struct {
	SubnetCidrMask     int
	EnableDNSHostnames bool
	EnableDNSSupport   bool
}

// GetEnvironmentConfig 環境名から設定を取得
func GetEnvironmentConfig(env string) (*EnvironmentConfig, error) {
	configs := map[string]*EnvironmentConfig{
		"dev": {
			Name:              "development",
			VpcCidr:           "10.0.0.0/16",
			MaxAzs:            2,
			EnableNATGateway:  true,
			EnableVPCFlowLogs: false,
			AllowSSHAccess:    true,
			RestrictedCIDRs:   []string{"10.0.0.0/8"}, // 開発環境では内部ネットワークのみ
			Tags: map[string]string{
				"Environment": "development",
				"Project":     "PracticeService",
				"Owner":       "DevTeam",
				"CostCenter":  "Development",
			},
		},
		"staging": {
			Name:              "staging",
			VpcCidr:           "10.1.0.0/16",
			MaxAzs:            2,
			EnableNATGateway:  true,
			EnableVPCFlowLogs: true,
			AllowSSHAccess:    false,
			RestrictedCIDRs:   []string{"10.1.0.0/16"},
			Tags: map[string]string{
				"Environment": "staging",
				"Project":     "PracticeService",
				"Owner":       "DevOpsTeam",
				"CostCenter":  "Testing",
			},
		},
		"prod": {
			Name:              "production",
			VpcCidr:           "10.2.0.0/16",
			MaxAzs:            3, // 本番環境は3AZ
			EnableNATGateway:  true,
			EnableVPCFlowLogs: true,
			AllowSSHAccess:    false,
			RestrictedCIDRs:   []string{"10.2.0.0/16"},
			Tags: map[string]string{
				"Environment": "production",
				"Project":     "PracticeService",
				"Owner":       "ProductionTeam",
				"CostCenter":  "Production",
				"Backup":      "Required",
			},
		},
	}

	config, exists := configs[env]
	if !exists {
		return nil, fmt.Errorf("unknown environment: %s. Available environments: dev, staging, prod", env)
	}

	return config, nil
}

// GetNetworkConfig ネットワーク固有の設定を取得
func GetNetworkConfig(env string) *NetworkConfig {
	// 全環境共通のネットワーク設定
	return &NetworkConfig{
		SubnetCidrMask:     24, // /24 サブネット
		EnableDNSHostnames: true,
		EnableDNSSupport:   true,
	}
}

// ValidateEnvironment 環境名が有効かチェック
func ValidateEnvironment(env string) bool {
	validEnvs := []string{"dev", "staging", "prod"}
	for _, validEnv := range validEnvs {
		if env == validEnv {
			return true
		}
	}
	return false
}

// GetAvailableEnvironments 利用可能な環境一覧を取得
func GetAvailableEnvironments() []string {
	return []string{"dev", "staging", "prod"}
}
