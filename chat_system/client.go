package main

import (
	"bufio"
	context "context"
	"strings"

	"fmt"
	"log"
	"os"
	pb "p1/chat"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var user_id int32
var current_state int32
var likeMap = make(map[int32]int32)
var wg sync.WaitGroup
var StreamLock sync.Mutex
var stream pb.Chat_ChatRoomClient

func main() {
	/****** Some global variables ******/
	var input_func string
	var username string // username
	currentgroup := ""  // currentgroup
	newgroup := ""
	request := ""
	user_id = -1
	var conn *grpc.ClientConn
	Connection := false
	current_state = 0
	StreamLock.Lock()
	wg.Add(1)
	go receiving()
	showCommand()
	for {
		// Read from keyboard
		reader := bufio.NewReader(os.Stdin)
		Myinput, _ := reader.ReadString('\n')
		input_func = Myinput[:len(Myinput)-1] // strip trailing '\n'
		arguments := strings.Split(input_func, " ")
		if len(arguments) != 2 {
			if arguments[0] != "a" && arguments[0] != "v" && arguments[0] != "p" && arguments[0] != "q" && arguments[0] != "data" && arguments[0] != "222" {
				println("Invalid input, please enter the correct format:")
				showCommand()
				continue
			}
		}
		/****** Switch loop to choose function ******/
		switch arguments[0] {
		case "c":
			if Connection {
				println("Lock 101")
				user_id = -1
				currentgroup = ""
				current_state = 0
				StreamLock.Lock()
			}
			TempConn, err := grpc.Dial(arguments[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatal("dialing:", err)
			}
			conn = TempConn
			client := pb.NewChatClient(TempConn)
			ctx := context.Background()
			TempStream, errrrr := client.ChatRoom(ctx)
			for errrrr != nil {
				TempStream, errrrr = client.ChatRoom(ctx)
				// sleep for 1 second
				time.Sleep(500 * time.Millisecond)
				println("retrying to connect to server")
			}
			stream = TempStream
			StreamLock.Unlock()
			Connection = true
			println("Successfully connected to server on " + arguments[1] + "!")
		case "u":
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			// Read from keyboard
			login_type := "u"
			if user_id != -1 {
				login_type = "change"
			}
			login := &pb.ClientRequest{Request: login_type, UserName: arguments[1], OriGroupName: currentgroup, UserId: user_id}
			// set username
			username = arguments[1]
			if err := stream.Send(login); err != nil {
				log.Fatal(err)
			}
			println("")
		/****** Join Group & Switch Group ******/
		case "j":
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state < 1 {
				fmt.Print("Please login first to join a group\n")
				continue
			}
			// Use join logic if first join
			// Read from keyboard
			// Set up currentgroup
			if currentgroup != "" {
				newgroup = arguments[1]
				request = "s"
			} else {
				currentgroup = arguments[1]
				request = "j"
			}
			// Set up arguments for RPC call
			joining := &pb.ClientRequest{Request: request, UserName: username, OriGroupName: currentgroup, NewGroupName: newgroup, UserId: user_id}
			if request == "s" {
				currentgroup = newgroup
			}
			// Do RPC call
			if err := stream.Send(joining); err != nil {
				log.Fatal(err)
			}
			current_state = 2
			println("")
		/****** Append to a chat ******/
		case "a":
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state < 2 {
				fmt.Print("Please join a group to append message\n")
				continue
			}
			// Read from keyboard
			fmt.Print("Please enter the message you want to send: ")
			reader := bufio.NewReader(os.Stdin)
			AppendMessage, _ := reader.ReadString('\n')
			AppendMessage = AppendMessage[:len(AppendMessage)-1]
			appending := &pb.ClientRequest{Request: "a", UserName: username, Content: AppendMessage, UserId: user_id, OriGroupName: currentgroup}
			// Do RPC call
			if err := stream.Send(appending); err != nil {
				log.Fatal(err)
			}
			println("")
		/****** Like a message ******/
		case "l":
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state < 2 {
				fmt.Print("Please join a group to like message\n")
				continue
			}
			// Read from keyboard
			// Get the message id for liked message
			inputInt, err := strconv.Atoi(arguments[1])
			if err != nil {
				log.Fatal(err)
			}
			likeMessageId, ok := likeMap[int32(inputInt)]
			if !ok {
				println("Please enter a valid message id")
				continue
			}
			// Set up arguments for RPC call
			liking := &pb.ClientRequest{Request: "l", OriGroupName: currentgroup, LikeNumber: likeMessageId, UserName: username, UserId: user_id}
			// Do RPC call
			if errr := stream.Send(liking); errr != nil {
				log.Fatal(errr)
			}
			println("")
		case "r": /****** Dislike a message ******/
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state < 2 {
				fmt.Print("Please join a group to unlike message\n")
				continue
			}
			// Get the message id for liked message
			input_int, err := strconv.Atoi(arguments[1])
			dislikeMessageId, ok := likeMap[int32(input_int)]
			if !ok {
				println("Please enter a valid message id")
				continue
			}
			if err != nil {
				log.Fatal(err)
			}
			// Set up arguments for RPC call
			disliking := &pb.ClientRequest{Request: "r", OriGroupName: currentgroup, LikeNumber: dislikeMessageId, UserId: user_id, UserName: username}
			// Do RPC call
			if errr := stream.Send(disliking); errr != nil {
				log.Fatal(errr)
			}
			println("")
		case "p": /****** Print History ******/
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state < 2 {
				fmt.Print("Please join a group to show history message\n")
				continue
			}
			// Set up arguments for RPC call
			printing := &pb.ClientRequest{Request: "p", OriGroupName: currentgroup}
			println("")
			// Do RPC call
			if err := stream.Send(printing); err != nil {
				log.Fatal(err)
			}
		case "q": /****** Quit ******/
			if Connection == false {
				fmt.Print("Please connect to a server firstly\n")
				continue
			}
			if current_state == 0 {
				println("i wanna break")
				current_state = 3
				break
			}
			// Set up arguments for RPC call
			quiting := &pb.ClientRequest{Request: "q", UserName: username, UserId: user_id, OriGroupName: currentgroup}
			println("")
			// Do RPC call
			if err := stream.Send(quiting); err != nil {
				log.Fatal(err)
			}
			wg.Wait() // wait for server response
			current_state = 3
			break
		case "v":
			view := &pb.ClientRequest{Request: "v"}
			if errr := stream.Send(view); errr != nil {
				log.Fatal(errr)
			}
		case "printdata":
			data := &pb.ClientRequest{Request: "data"}
			if errr := stream.Send(data); errr != nil {
				log.Fatal(errr)
			}
		case "inserttest":
			test := &pb.ClientRequest{Request: "test", OriGroupName: currentgroup, Replica: 3, Timestamp: 3, UserName: username, UserId: user_id, Content: "test"}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "like":
			lamport, _ := strconv.Atoi(arguments[1])
			replica, _ := strconv.Atoi(arguments[2])
			like, _ := strconv.Atoi(arguments[3])
			now := time.Now().Format(time.RFC3339)
			test := &pb.ClientRequest{Request: "UpdateLike", OriGroupName: currentgroup, Content: now, UserName: username,
				UserId: user_id, Timestamp: int32(lamport), Replica: int32(replica), LikeNumber: int32(like)}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "old":
			lamport, _ := strconv.Atoi(arguments[1])
			replica, _ := strconv.Atoi(arguments[2])
			like, _ := strconv.Atoi(arguments[3])
			now := time.Now().Add(-30 * time.Second)
			nowString := now.Format(time.RFC3339)
			test := &pb.ClientRequest{Request: "UpdateLike", OriGroupName: currentgroup, Content: nowString, UserName: username,
				UserId: user_id, Timestamp: int32(lamport), Replica: int32(replica), LikeNumber: int32(like)}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "member":
			input_int, _ := strconv.Atoi(arguments[1])
			input_int_2, _ := strconv.Atoi(arguments[2])
			mm := ", test1" + arguments[1] + ", test2" + arguments[1] + ", test3" + arguments[1]
			test := &pb.ClientRequest{Request: "SendGroupInfo", OriGroupName: currentgroup,
				Content: mm, Replica: int32(input_int), Timestamp: int32(input_int_2)}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "member2":
			input_int, _ := strconv.Atoi(arguments[1])
			mm := ", test0 ,"
			test := &pb.ClientRequest{Request: "SendGroupInfo", OriGroupName: currentgroup, Content: mm, Replica: int32(input_int)}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "sendSame":
			currentgroup = "test"
			test := &pb.ClientRequest{Request: "test", OriGroupName: currentgroup, Content: "mMysg", UserName: username,
				UserId: user_id, Timestamp: 3, Replica: 4, LikeNumber: 1}
			if errr := stream.Send(test); errr != nil {
				log.Fatal(errr)
			}
		case "data":
			data := &pb.ClientRequest{Request: "data"}
			if errr := stream.Send(data); errr != nil {
				log.Fatal(errr)
			}
		case "send":
			Entropy := &pb.ClientRequest{Request: "Entropy"}
			if errr := stream.Send(Entropy); errr != nil {
				log.Fatal(errr)
			}
		case "000":
			Entropy := &pb.ClientRequest{Request: "000"}
			if errr := stream.Send(Entropy); errr != nil {
				log.Fatal(errr)
			}
		case "111":
			mapp := &pb.ClientRequest{Request: "InspectMap"}
			if errr := stream.Send(mapp); errr != nil {
				log.Fatal(errr)
			}
		case "222":
			mapp := &pb.ClientRequest{Request: "Entropy"}
			if errr := stream.Send(mapp); errr != nil {
				log.Fatal(errr)
			}
		default:
			println("Invalid input, please enter the correct format:")
			showCommand()
		}
		if current_state == 3 {
			println("i wanna break")
			break
		}
	}
	/****** Quit client process ******/
	println("Goodbye!")
	if errrr := conn.Close(); errrr != nil {
		log.Fatal(errrr)
	}
}

func receiving() {
	for {
		StreamLock.Lock()
		StreamLock.Unlock()
		resp, err := stream.Recv()

		if err != nil {
			log.Fatalf("cannot receive %v", err)
		}
		if resp.Request == "Denial" {
			println(resp.Content)
		}
		if resp.Request == "View" {
			println(resp.Content)
		}
		if resp.Request == "print" {
			if resp.Content != "" && resp.Username != "" {
				fmt.Printf("%s: ", resp.Username)
				fmt.Printf("%s\t", resp.Content)
				if resp.Like != 0 {
					fmt.Printf("Likes: %d", resp.Like)
				}
				fmt.Printf("\n")
			}
		} else if resp.Request == "login" {
			user_id = resp.UserId
			if current_state == 0 {
				current_state = 1
			}
			fmt.Printf("Your User Id: %v, User Name: %v\n", user_id, resp.Username)
		} else if resp.Request == "store" {
			likeMap[resp.LocalMessage] = resp.MsgId
			//fmt.Printf("%v store at position: %v\n", resp.Content, resp.LocalMessage)
			if resp.Content != "" && resp.Username != "" {
				fmt.Printf("%s: ", resp.Username)
				fmt.Printf("%s\t", resp.Content)
				if resp.Like != 0 {
					fmt.Printf("                     Likes: %d", resp.Like)
				}
				fmt.Printf("\n")
			}
		} else if resp.Request == "bye" {
			fmt.Printf("%s\t", resp.Content)
			wg.Done()
			return
		}

	}
}
func showCommand() {
	println("connect:         c localhost:12000")
	println("login: 	         u Amy")
	println("join group:      j group1")
	println("append message:  a")
	println("like:            l 5")
	println("remove like:     r 5")
	println("print history:   p")
	println("view:            v")
	println("quit:            q")
	println("show all data:            data")
	println("if the screen does not refresh after partition:")
	println("Please wait and then type:  222")
	println("The network may delay for a while")
	println("or type:  p to print the history")
}
