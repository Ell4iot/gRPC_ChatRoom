package chat

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// linked list structure for user

func NewServer() *Server {
	GroupMap := make(map[string]*Group)
	sm := &Server{GroupMap: GroupMap, UserCount: 0}
	return sm
}

func (sm *Server) ChatRoom(ct Chat_ChatRoomServer) error {
	// store the current client's userID
	var ThisUserid int32
	var ThisName string
	var thisClientGroup *Group
	var ReplicaName string
	IamAserver := false
	// get to know the server number
	args := os.Args[1:]
	serverNum, _ := strconv.Atoi(args[1])
	println("server number: ", serverNum)

	// Perform anti-entropy to other 4 servers.

	// Below is the logic of the server processing the request
	for {
		message, err := ct.Recv()
		println("----------------------server receives a request-------------------------")
		if err != nil {
			log.Printf("recv err: %v", err)
			if IamAserver {
				println("Replica connection stops")
				println("Disconnected server: " + ReplicaName)
				sm.StreamSet[ReplicaName].isOnline = 0
			} else {
				println("user connection stops")
				if thisClientGroup != nil {
					println("user leaves the group because of disconnection")
					// monitor whether the connection is closed or not
					delete(thisClientGroup.UserList, ThisUserid)
					// user left, group participants list should change
					sm.DeleteUsername(thisClientGroup, ThisName)
					// broadcast the group info to all the clients
					sm.ClientBroadcast(thisClientGroup)
					// flag == 0 indicates only send my server's info of the group to all servers
					quitGroup := &ClientRequest{Request: "SendGroupInfo", UserName: "",
						Content: thisClientGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: thisClientGroup.GroupName, LikeNumber: 0,
						FromServer: sm.StringMe, Timestamp: thisClientGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
						GroupTimestamp: thisClientGroup.MaxLamport}
					sm.Mu.Lock()
					sm.ReplicaBroadcast(quitGroup)
					sm.Mu.Unlock()
				}

			}
			break
		}
		switch message.Request {
		case "u":
			println("User login")
			ThisName = message.UserName
			reply, id := sm.UserLogin(message)
			ThisUserid = id
			if err := ct.Send(reply); err != nil {
				log.Printf("send error %v", err)
			}
			println("")
		case "change":
			println("Change identity")
			ThisName = message.UserName
			reply, theGroup := sm.ChangeIdentity(message)
			if err := ct.Send(reply); err != nil {
				log.Printf("send error %v", err)
			}
			if theGroup != nil {
				sm.ClientBroadcast(theGroup)
				groupListChange := &ClientRequest{Request: "SendGroupInfo", UserName: "",
					Content: theGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: theGroup.GroupName, LikeNumber: 0,
					FromServer: sm.StringMe, Timestamp: theGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
					GroupTimestamp: theGroup.MaxLamport}
				sm.Mu.Lock()
				sm.ReplicaBroadcast(groupListChange)
				sm.Mu.Unlock()
			}
			println("")
		case "j":
			//TODO: SEND GROUP INFO BROADCASTING
			println("join group")
			theGroup := sm.Join(message, ct)
			thisClientGroup = theGroup
			sm.ClientBroadcast(theGroup)
			// FromServer is the server that this message belongs to
			groupListChange := &ClientRequest{Request: "SendGroupInfo", UserName: "",
				Content: theGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: message.OriGroupName, LikeNumber: 0,
				FromServer: sm.StringMe, Timestamp: theGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
				GroupTimestamp: theGroup.MaxLamport}
			sm.Mu.Lock()
			sm.ReplicaBroadcast(groupListChange)
			sm.Mu.Unlock()
			println("")
		case "a":
			println("append message")
			theGroup, theMessage := sm.AppendMessage(message)
			sm.ClientBroadcast(theGroup)
			activity := &ClientRequest{Request: "UpdateMsg", UserName: message.UserName,
				Content: message.Content, OriGroupName: message.OriGroupName, LikeNumber: 0,
				Replica: sm.Me, Timestamp: theMessage.Lamport, FromServer: sm.StringMe,
				MapIndex: theGroup.ReplicaInfo[sm.StringMe].MsgTimestamp, GroupTimestamp: theGroup.MaxLamport}
			sm.Mu.Lock()
			sm.ReplicaBroadcast(activity)
			sm.Mu.Unlock()
			println("")
		case "l":
			println("like message")
			theGroup, indicator, theTime, ok, msgLamport, belongReplica, index := sm.LikeMessage(message)
			if ok == true {
				sm.ClientBroadcast(theGroup)
				activity := &ClientRequest{Request: "UpdateLike", UserName: message.UserName,
					Content: theTime, OriGroupName: message.OriGroupName, LikeNumber: 1,
					Replica: belongReplica, Timestamp: msgLamport, FromServer: sm.StringMe,
					MapIndex: index, GroupTimestamp: theGroup.MaxLamport}
				sm.Mu.Lock()
				sm.ReplicaBroadcast(activity)
				sm.Mu.Unlock()
			} else {
				reply := ServerResponse{Request: "Denial", Content: indicator}
				if err := ct.Send(&reply); err != nil {
					log.Printf("send error %v", err)
				}
			}
		case "r":
			println("unlike message")
			theGroup, indicator, theTime, ok, msgLamport, belongReplica, index := sm.UnlikeMessage(message)
			if ok == true {
				sm.ClientBroadcast(theGroup)
				activity := &ClientRequest{Request: "UpdateLike", UserName: message.UserName,
					Content: theTime, OriGroupName: message.OriGroupName, LikeNumber: -1,
					Replica: belongReplica, Timestamp: msgLamport, FromServer: sm.StringMe, MapIndex: index,
					GroupTimestamp: theGroup.MaxLamport}
				sm.Mu.Lock()
				sm.ReplicaBroadcast(activity)
				sm.Mu.Unlock()
			} else {
				reply := ServerResponse{Request: "Denial", Content: indicator}
				if err := ct.Send(&reply); err != nil {
					log.Printf("send error %v", err)
				}
			}
			println("")
		case "p":
			println("show history")
			sm.ShowHistory(message.OriGroupName, &ct)
		case "s":
			println("switch group")
			oldGroup, newGroup := sm.Switch(message)
			thisClientGroup = newGroup
			sm.ClientBroadcast(newGroup)
			sm.ClientBroadcast(oldGroup)
			OldgroupListChange := &ClientRequest{Request: "SendGroupInfo", UserName: "",
				Content: oldGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: oldGroup.GroupName, LikeNumber: 0,
				FromServer: sm.StringMe, Timestamp: oldGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
				GroupTimestamp: oldGroup.MaxLamport}
			sm.Mu.Lock()
			sm.ReplicaBroadcast(OldgroupListChange)
			sm.Mu.Unlock()

			NewgroupListChange := &ClientRequest{Request: "SendGroupInfo", UserName: "",
				Content: newGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: newGroup.GroupName, LikeNumber: 0,
				FromServer: sm.StringMe, Timestamp: newGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
				GroupTimestamp: newGroup.MaxLamport}
			sm.Mu.Lock()
			sm.ReplicaBroadcast(NewgroupListChange)
			sm.Mu.Unlock()
			println("")
		case "q":
			println("user quiting")
			bye, theGroup := sm.QuitSession(message)
			if theGroup != nil {
				// Will be delted it in the upper block of checking connection code
				sm.ClientBroadcast(theGroup)
				quitGroup := &ClientRequest{Request: "SendGroupInfo", UserName: "",
					Content: theGroup.ReplicaInfo[sm.StringMe].UserList, OriGroupName: theGroup.GroupName, LikeNumber: 0,
					FromServer: sm.StringMe, Timestamp: theGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp,
					GroupTimestamp: theGroup.MaxLamport}
				sm.Mu.Lock()
				sm.ReplicaBroadcast(quitGroup)
				sm.Mu.Unlock()
			}
			reply := ServerResponse{Request: "bye", Content: bye}
			if err := ct.Send(&reply); err != nil {
				log.Printf("send error %v", err)
			}
			println("")
		case "v":
			println("view")
			returnS := "View: "
			for server, sNode := range sm.StreamSet {
				nnow := time.Now()
				// if time difference is less than 5 seconds, then it is alive
				if nnow.Sub(sNode.LastTime).Seconds() < 3 {
					returnS += "Replica." + server + " "
				}
			}
			println(returnS)
			view := &ServerResponse{Request: "View", Content: returnS}
			if err := ct.Send(view); err != nil {
				log.Printf("send error %v", err)
			}
		case "UpdateMsg":
			flush := sm.Merge(message, 0)
			if flush != nil {
				println("Refresh")
				sm.FlushScreen(flush)
			}

		case "UpdateLike":
			flush := sm.Merge(message, 1)
			if flush != nil {
				println("Refresh")
				sm.FlushScreen(flush)
			}
		case "SendGroupInfo":
			flush := sm.Merge(message, 2)
			if flush != nil {
				println("Refresh")
				sm.FlushScreen(flush)
			}
		case "Entropy":
			//sm.AntiEntropy()
			sm.Mu.Lock()
			sm.Entropy()
			sm.Mu.Unlock()
		case "ReplicaSetUp":
			println("ReplicaSetUp Name: " + message.UserName)
			ReplicaName = message.UserName
			IamAserver = true
		case "AllGroup":
			nn := time.Now()
			if sm.StreamSet[message.FromServer] != nil {
				sm.StreamSet[message.FromServer].LastTime = nn
			}
			sm.AllGroup(ct)
		case "CheckGroup":
			nn := time.Now()
			if sm.StreamSet[message.UserName] != nil {
				sm.StreamSet[message.FromServer].LastTime = nn
			}
			sm.CheckGroup(message.Content)
		case "CheckForMerge":
			sm.CheckForMerge(message)
		case "data":
			sm.PrintData()
		case "EntropyMsg":
			sm.Merge(message, 0)
		case "test":
			sm.test(message)
		case "docker":
			println(message.Content)
		case "liketest":
			sm.testLike(message)
		case "000":
			sm.testMerge()
		case "InspectMap":
			sm.InspectMap()
		}
	}
	return nil
}

func (sm *Server) ClientBroadcast(theGroup *Group) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	if theGroup == nil {
		return
	}
	println("broadcasting message to every client in the group")
	var i int32
	for _, client := range theGroup.UserList {
		println("broadcasting to " + client.UserName)
		node := theGroup.MessageList.TenStart
		for i = 0; i < 12; i++ {
			sending := ServerResponse{}
			if i == 0 {
				content := "CS2510 " + theGroup.GroupName
				sending = ServerResponse{Request: "print", Username: "Group", Content: content}
			} else if i == 1 {
				TotalUserNames := NameMerge(theGroup)
				sending = ServerResponse{Request: "print", Username: "Participants", Content: TotalUserNames}
			} else {
				format := strconv.Itoa(int(i - 1))
				if node != nil {
					format = format + ". " + node.Message.Creator
					sending = ServerResponse{Request: "store", Username: format,
						Content: node.Message.Content, Like: node.Message.TotalLike, MsgId: node.Message.MsgId, LocalMessage: i - 1}
					node = node.Next
				} else {
					break
				}
			}
			if err := client.context.Send(&sending); err != nil {
				log.Printf("send error %v", err)
			}
		}
	}
}

func (sm *Server) UserLogin(message *ClientRequest) (*ServerResponse, int32) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	sm.UserCount++
	//sm.UserCount is the user id
	//sm.TotalUser.Append(message.sameUserNameCount, sm.UserCount)
	reply := &ServerResponse{Request: "login", UserId: sm.UserCount, Username: message.UserName}
	fmt.Printf("%v has logged in....\n", message.UserName)
	return reply, sm.UserCount
}

func (sm *Server) ChangeIdentity(message *ClientRequest) (*ServerResponse, *Group) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	//sm.UserCount is the user id
	theGroup := sm.GroupMap[message.OriGroupName]
	if theGroup != nil {
		fmt.Printf("delete previous role, pre ID: %v\n", message.UserId)
		previousName := theGroup.UserList[message.UserId].UserName
		theGroup.UserList[message.UserId].UserName = message.UserName
		sm.DeleteUsername(theGroup, previousName)
		sm.AddUsername(theGroup, message.UserName)
	} else {
		println("no group assigned before")
	}
	reply := &ServerResponse{Request: "login", UserId: message.UserId, Username: message.UserName}
	println("ChangeIdentity() Done\n")
	return reply, theGroup
}

func (sm *Server) AppendMessage(message *ClientRequest) (*Group, *Message) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	theGroup := sm.GroupMap[message.OriGroupName]
	// update the group lamport timestamp\
	theGroup.MaxLamport++
	// Initialize the message node
	thisMessage := newMessage(theGroup.MaxLamport, sm.Me, message.UserName, message.Content, 0)
	newNode := theGroup.MessageList.Append()
	// append the newNode to the slice
	theGroup.ReplicaInfo[sm.StringMe].MsgTimestamp++
	theGroup.ReplicaInfo[sm.StringMe].MessageLamport[theGroup.ReplicaInfo[sm.StringMe].MsgTimestamp] = newNode

	thisMessage.MsgId = theGroup.MessageList.Count
	newNode.Message = thisMessage
	println("251")
	if theGroup.MessageList.Count == 1 {
		println("255 first message")
		theGroup.MessageList.TenStart = newNode

	} else if theGroup.MessageList.Count > 10 {
		println("moving latest ten message start point")
		theGroup.MessageList.TenStart = theGroup.MessageList.TenStart.Next
	}

	println("AppendMessage() Done\n")
	return theGroup, thisMessage
}

func (sm *Server) Join(message *ClientRequest, ct Chat_ChatRoomServer) *Group {
	// create group if group does not exist
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	theGroup, ok := sm.GroupMap[message.OriGroupName]
	println("268")
	if !ok {
		// create new group
		theGroup = sm.CreateGroup(message.OriGroupName, 0)
		sm.GroupMap[message.OriGroupName] = theGroup
	}
	println("277")
	// append the username to the current group
	theGroup.UserList[message.UserId] = &User{UserName: message.UserName, context: ct}
	sm.AddUsername(theGroup, message.UserName)
	println("Join() Done\n")
	return theGroup
}

func (sm *Server) Switch(message *ClientRequest) (*Group, *Group) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	// remove from old group
	oldGroup := sm.GroupMap[message.OriGroupName]
	fmt.Printf("user: %v id: %v want to be deleted\n", message.UserName, message.UserId)
	// delete it from previous group
	user := oldGroup.UserList[message.UserId]
	sm.DeleteUsername(oldGroup, user.UserName)
	delete(oldGroup.UserList, message.UserId)

	// create group if group does not exist
	newGroup, ok := sm.GroupMap[message.NewGroupName]
	if !ok {
		newGroup = sm.CreateGroup(message.NewGroupName, 0)
		sm.GroupMap[message.NewGroupName] = newGroup
	}
	// append the username to the current group
	newGroup.UserList[message.UserId] = user
	sm.AddUsername(newGroup, message.UserName)
	println("Switch(): Done\n")
	return oldGroup, newGroup
}

func (sm *Server) LikeMessage(message *ClientRequest) (*Group, string, string, bool, int32, int32, int32) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	var NowString string
	var belongReplica int32
	var index int32
	success := false
	msgLamport := int32(0)
	theGroup := sm.GroupMap[message.OriGroupName]
	fmt.Printf("group: %v\n", message.OriGroupName)
	indicator := ""
	if message.LikeNumber > theGroup.MessageList.Count || message.LikeNumber == 0 {
		println("illegal like behavior: out of bounds")
		indicator = "illegal like request: out of bounds"
	} else {
		println("like number is legal")
		// find the message
		head := theGroup.MessageList.Head
		for head != nil {
			if head.Message.MsgId == message.LikeNumber {
				belongReplica = head.Message.Replica
				// find the message
				println("message to like found")
				if head.Message.Creator == message.UserName {
					// owner can't like it
					println("illegal like request: You can't like your own message")
					indicator = "illegal like request: You can't like your own message"
				} else {
					// not owner
					like, ok := head.Message.UserLike[message.UserName]
					if ok {
						if like <= 0 {
							NowString, index = sm.PerformingLike(theGroup, 1, head.Message, message.UserName)
							success = true
							msgLamport = head.Message.Lamport
						} else {
							println("illegal like request: You have already liked it")
							indicator = "illegal like request: You have already liked it"
						}
					} else {
						NowString, index = sm.PerformingLike(theGroup, 1, head.Message, message.UserName)
						success = true
						msgLamport = head.Message.Lamport
					}
				}
				break
			}
			head = head.Next
		}
	}

	return theGroup, indicator, NowString, success, msgLamport, belongReplica, index
}

func (sm *Server) UnlikeMessage(message *ClientRequest) (*Group, string, string, bool, int32, int32, int32) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	var belongReplica int32
	theGroup := sm.GroupMap[message.OriGroupName]
	indicator := ""
	msgLamport := int32(0)
	var NowString string
	var index int32
	success := false
	if message.LikeNumber > theGroup.MessageList.Count || message.LikeNumber == 0 {
		println("out of bounds")
		indicator = "illegal unlike request: out of bounds"
	} else {
		println("unlike number is legal")
		// find the message
		head := theGroup.MessageList.Head
		for head != nil {
			fmt.Printf("message id: %v, message to unlike； %v", head.Message.MsgId, message.LikeNumber)
			if head.Message.MsgId == message.LikeNumber {
				// find the message
				belongReplica = head.Message.Replica
				println("message to unlike found")
				if head.Message.Creator == message.UserName {
					// owner can't unlike it
					println("illegal like request: You can't unlike your own message")
					indicator = "illegal like request: You can't unlike your own message"
				} else {
					like, ok := head.Message.UserLike[message.UserName]
					if ok {
						if like == 1 {
							NowString, index = sm.PerformingLike(theGroup, -1, head.Message, message.UserName)
							success = true
							msgLamport = head.Message.Lamport
						} else {
							indicator = "illegal unlike request: You have not liked it"
						}
					} else {
						indicator = "illegal unlike request: You have not liked it"
					}
				}
				break
			}
			head = head.Next
		}
	}
	return theGroup, indicator, NowString, success, msgLamport, belongReplica, index
}

func (sm *Server) ShowHistory(GroupName string, ct *Chat_ChatRoomServer) {
	// create group if group does not exist
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	println("show history")
	theGroup := sm.GroupMap[GroupName]
	node := theGroup.MessageList.Head
	reply := ServerResponse{}
	var i = int32(0)
	counter := i
	for {
		if i > 1 && node == nil {
			break
		}
		if i == 0 {
			content := "CS2510 " + theGroup.GroupName + " chat history"
			reply = ServerResponse{Request: "print", Username: "Group", Content: content}
		} else if i == 1 {
			TotalUserNames := NameMerge(theGroup)
			reply = ServerResponse{Request: "print", Username: "Participants", Content: TotalUserNames}
		} else {
			format := strconv.Itoa(int(i - 1))
			format = format + ". " + node.Message.Creator
			reply = ServerResponse{Request: "store", Username: format, Content: node.Message.Content,
				Like: node.Message.TotalLike, MsgId: node.Message.MsgId, LocalMessage: i - 1}
			node = node.Next
			counter++
		}
		if err := (*ct).Send(&reply); err != nil {
			log.Printf("send error %v", err)
		}
		i++
	}
}

func (sm *Server) QuitSession(message *ClientRequest) (string, *Group) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	//sm.UserCount is the user id
	println("client request to quit.....")
	//sm.TotalUser.Delete(message.UserId)
	oldGroup, ok := sm.GroupMap[message.OriGroupName]
	if ok {
		// Remeber not to delete it on the server side
		println("delete client ID from group")
		println(oldGroup.ReplicaInfo[sm.StringMe].UserList)
	} else {
		oldGroup = nil
	}

	goodbyeMessage := "UserID:" + strconv.Itoa(int(message.UserId)) + " left session"
	return goodbyeMessage, oldGroup
}
