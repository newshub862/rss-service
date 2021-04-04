package model

// Config - application configuration structure
type Config struct {
	Driver           string `json:"driver"`
	ConnectionString string `json:"connection_string"`
	DbHost           string `json:"db_host"`
	DbName           string `json:"db_name"`
	DbUser           string `json:"db_user"`
	DbPassword       string `json:"db_password"`
	DbPort           int    `json:"db_port"`
	UpdateMinutes    int    `json:"update_minutes"`
	ArticlesMaxCount int    `json:"articles_max_count"`
}
