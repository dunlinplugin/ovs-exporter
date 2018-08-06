package main

//This file contains the JSON response object (structs)

//Written by Megyo @ LeanNet 

//import "time"

type Flow struct {
    Cookie    	string    	`json:"cookie"`
    Duration  	string   	`json:"duration"`
    Table     	string 		`json:"table"`
    Packets   	string		`json:"packets"`
    Bytes     	string		`json:"bytes"`
    IdleTimeout string		`json:"idletimeout"` 
    IdleAge		string		`json:"idleage"`
    Priority	string		`json:"proirity"`
    Match		string		`json:"match"`
    Action		string		`json:"action"`
    ServiceName string      `json:"servicename"`
    ServiceIP   string      `json:"serviceip"`
    ServiceNamespace string `json:"servicenamespace"`
    NodeName    string      `json:"nodename"`
    NodeIP      string      `json:"nodeip"`
}

type Flows []Flow

type Port struct {
    PortNumber   string    	`json:"portnumber"`
    RxPackets  	 string   	`json:"rxpackets"`
    TxPackets    string 	`json:"txpackets"`
    RxBytes   	 string		`json:"rxbytes"`
    TxBytes      string		`json:"txbytes"`
    RxDrops	 	 string		`json:"rxdrops"` 
    TxDrops		 string		`json:"txdrops"`
    RxErrors	 string		`json:"rxerrors"`
    TxErrors	 string		`json:"txerrors"`
    RxFrameErr	 string		`json:"rxframeerr"`
    RxOverruns	 string		`json:"rxovverruns"`
    RxCrcErrors	 string		`json:"rxcrcerrors"`
    TxCollisions string		`json:"txcollisions"`
    PodName      string     `json:"podname"`
    PodNamespace string     `json:"podnamespace"`
    PodIP        string     `json:"podip"`
}

type Ports []Port

type Group struct {
    GroupId   string    `json:"groupid"`
    GroupType string   	`json:"grouptype"`
    Buckets   []Bucket 	`json:"buckets"` 
    Duration  string	`json:"duration"`
    Bytes     string	`json:"bytes"`
    Packets   string    `json:"packets"`
    ServiceName string  `json:"servicename"`
    ServiceIP   string  `json:"serviceip"`
    ServiceNamespace string      `json:"servicenamespace"`
}

type Bucket struct {
//    BucketId  string    `json:"bucketid"` // for now I see no real usage of BucketID
    Actions      string 	`json:"actions"` 
    Bytes        string		`json:"bytes"`
    Packets      string     `json:"packets"`
    PodName      string     `json:"podname"`
    PodNamespace string     `json:"podnamespace"`
    PodIP        string     `json:"podip"`
}

type Groups []Group

