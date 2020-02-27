package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type DB struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	UserId     string `json:"user_id"`
	Password   string `json:"password"`
	AreaName   string `json:"area_name"`
	ServerName string `json:"server_name"`
	RoleName   string `json:"role_name"`
}

func getAccounts() []Account {
	j, err := ioutil.ReadFile("account.json")
	if err != nil {
		log.Printf("Fail to ReadFile: %v\n", err)
	}
	db := &DB{}
	err = json.Unmarshal(j, db)
	if err != nil {
		log.Printf("Fail to Unmarshal: %v\n", err)
	}
	return db.Accounts
}
