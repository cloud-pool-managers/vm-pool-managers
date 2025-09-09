package models

type ServerConfig struct {
	Name           string   `toml:"Name"`
	MinServer      int      `toml:"MinServer"`
	MaxServer      int      `toml:"MaxServer"`
	ImageRef       string   `toml:"ImageRef"`
	FlavorRef      string   `toml:"FlavorRef"`
	UUID           string   `toml:"UUID"`
	Subnet         string   `toml:"Subnet"`
	SecurityGroups []string `toml:"SecurityGroups"`
	Keyname        string   `toml:"Keyname"`
}

type VolumeConfig struct {
	Size        int    `toml:"Size"`
	Description string `toml:"Description"`
	Name        string `toml:"Name"`
	Volumetype  string `toml:"Volumetype"`
}

type NetworkConfig struct {
	NetworkID string `toml:"NetworkID"`
}

// structure du fichier de configuration pour créer les servers, les metadatas et les volumes
type Config struct {
	Server   ServerConfig      `toml:"server"`
	Metadata map[string]string `toml:"metadata"`
	Volume   VolumeConfig      `toml:"volume"`
	Network  NetworkConfig     `toml:"network"`
}
