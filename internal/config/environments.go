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

// 🆕 ECSConfig ECS Fargate固有の設定
type ECSConfig struct {
	CPU                    int  // 256, 512, 1024, 2048, 4096
	Memory                 int  // 512, 1024, 2048, 4096, 8192
	DesiredCount           int  // 1, 2, 4
	MinCapacity            int  // Auto Scaling最小
	MaxCapacity            int  // Auto Scaling最大
	EnableServiceDiscovery bool // Service Discovery有効化
	EnableLogging          bool // CloudWatch Logs
	EnableFargateSpot      bool // Fargate Spot使用
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

// 🆕 GetECSConfig 環境別のECS設定を取得
func GetECSConfig(environment string) *ECSConfig {
	switch environment {
	case "dev":
		return &ECSConfig{
			CPU:                    256,
			Memory:                 512,
			DesiredCount:           1,
			MinCapacity:            1,
			MaxCapacity:            2,
			EnableServiceDiscovery: false,
			EnableLogging:          true,
			EnableFargateSpot:      true, // 開発環境はコスト削減
		}
	case "staging":
		return &ECSConfig{
			CPU:                    512,
			Memory:                 1024,
			DesiredCount:           2,
			MinCapacity:            1,
			MaxCapacity:            4,
			EnableServiceDiscovery: true,
			EnableLogging:          true,
			EnableFargateSpot:      true, // ステージングでもコスト削減
		}
	case "prod":
		return &ECSConfig{
			CPU:                    1024,
			Memory:                 2048,
			DesiredCount:           4,
			MinCapacity:            2,
			MaxCapacity:            10,
			EnableServiceDiscovery: true,
			EnableLogging:          true,
			EnableFargateSpot:      false, // 本番環境は安定性優先
		}
	default:
		// デフォルト設定（開発環境相当）
		return &ECSConfig{
			CPU:                    256,
			Memory:                 512,
			DesiredCount:           1,
			MinCapacity:            1,
			MaxCapacity:            2,
			EnableServiceDiscovery: false,
			EnableLogging:          true,
			EnableFargateSpot:      true,
		}
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

// 🆕 GetCPUMemoryCombinations 有効なCPU・メモリの組み合わせを取得
func GetCPUMemoryCombinations() map[int][]int {
	return map[int][]int{
		256:  {512, 1024, 2048},
		512:  {1024, 2048, 3072, 4096},
		1024: {2048, 3072, 4096, 5120, 6144, 7168, 8192},
		2048: {4096, 5120, 6144, 7168, 8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384},
		4096: {8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384, 17408, 18432, 19456, 20480, 21504, 22528, 23552, 24576, 25600, 26624, 27648, 28672, 29696, 30720},
	}
}

// 🆕 ValidateECSConfig ECS設定の妥当性を検証
func ValidateECSConfig(config *ECSConfig) error {
	validCombos := GetCPUMemoryCombinations()

	if validMemories, ok := validCombos[config.CPU]; ok {
		for _, validMemory := range validMemories {
			if config.Memory == validMemory {
				return nil // 有効な組み合わせ
			}
		}
		return fmt.Errorf("invalid CPU/Memory combination: CPU=%d, Memory=%d", config.CPU, config.Memory)
	}

	return fmt.Errorf("invalid CPU value: %d", config.CPU)
}
