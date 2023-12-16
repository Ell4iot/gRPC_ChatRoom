package main

import (
	"bufio"
	"log"
	"net"
	"os"
	pb "p1/chat"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

// read command line arguments
func main() {
	// open a new file for logging
	args := os.Args[1:]
	serverStr := args[1]
	serverNum, _ := strconv.Atoi(serverStr)
	println("Starting gRPC server chat room")
	listener, err := net.Listen("tcp", "172.30.100.10"+serverStr+":12000")
	if err != nil {
		panic(err)
	}

	// create a gRPC server object
	chat_server := grpc.NewServer()
	myServerObj := pb.NewServer()
	tempMap := make(map[string]*pb.ServerConfig)
	myServerObj.StreamSet = tempMap
	myServerObj.Me = int32(serverNum)
	myServerObj.StringMe = serverStr

	file, err := os.Open("group.txt")
	if err != nil {
		file, _ = os.Create("group.txt")
		file.Close()
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			myServerObj.CreateGroup(scanner.Text(), int32(0))
		}
		file.Close()
	}

	var temp [4]string
	index := 0
	var i int32
	for i = 1; i <= 5; i++ {
		if i != myServerObj.Me {
			temp[index] = strconv.Itoa(int(i))
			index++
		}
	}

	go myServerObj.ReplicaSetUp(temp[0])
	go myServerObj.ReplicaSetUp(temp[1])
	go myServerObj.ReplicaSetUp(temp[2])
	go myServerObj.ReplicaSetUp(temp[3])
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			myServerObj.Mu.Lock()
			request := &pb.ClientRequest{Request: "AllGroup", FromServer: serverStr}
			myServerObj.ReplicaBroadcast(request)
			myServerObj.Mu.Unlock()
		}
	}()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			myServerObj.Mu.Lock()
			myServerObj.Entropy()
			myServerObj.Mu.Unlock()
		}
	}()

	//go myServerObj.Trying()
	// start the server
	pb.RegisterChatServer(chat_server, myServerObj)
	if err := chat_server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
