package chat

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"strconv"

	//"strconv"
	"time"
)

// ReplicaBroadcast sends a message to all replicas

func (sm *Server) Entropy() {
	// Send the history of this server to all other servers
	for _, group := range sm.GroupMap {
		println("Entropy(): sending what I got of group: " + group.GroupName)
		sm.SendMySaving(group)
	}
	println("Entropy(): done")
	return
}

func (sm *Server) SendMySaving(group *Group) {
	fmt.Printf("SendGroupInfo(): sending group: %v's info to all replicas\n", group.GroupName)
	for replica, info := range group.ReplicaInfo {
		println("SendMySaving() Requested server: " + replica)
		fmt.Printf("GropTime: %v\n", info.GroupNameTimestamp)
		fmt.Printf("MsgTime: %v\n", info.MsgTimestamp)
		fmt.Printf("LikeTime: %v\n", info.LikeTimestamp)
		SendMe := &ClientRequest{Request: "CheckForMerge", FromServer: replica, OriGroupName: group.GroupName,
			GroupTimestamp: info.GroupNameTimestamp, Timestamp: info.MsgTimestamp,
			Replica: sm.Me, LikeNumber: info.LikeTimestamp}
		sm.ReplicaBroadcast(SendMe)
	}
	println("SendMySaving(): successfully sent my group info to all replicas")
}

func (sm *Server) CheckForMerge(message *ClientRequest) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	// message.Fromserver - requested server info
	// message.OriGroupName - requested group name
	// message.GroupTimestamp - requested server groupName timestamp
	// message.Timestamp - requested server msg timestamp
	// message.Replica - requested server replica number
	// message.LikeNumber - requested server like timestamp
	theGroup := sm.GroupMap[message.OriGroupName]
	if theGroup == nil {
		return
	}
	sString := strconv.Itoa(int(message.Replica))
	fmt.Printf("CheckForMerge(): requested %v's info from %v\n", message.FromServer, sString)
	replyStream := sm.StreamSet[sString].Stream
	sm.StreamSet[sString].isOnline = 1
	fmt.Printf("receive group timestamp: %v, local group timestamp: %v\n",
		message.GroupTimestamp, theGroup.ReplicaInfo[message.FromServer].GroupNameTimestamp)
	if message.GroupTimestamp < theGroup.ReplicaInfo[message.FromServer].GroupNameTimestamp {
		println("CheckForMerge(): Need to update group info")
		updateGroup := &ClientRequest{Request: "SendGroupInfo", UserName: "",
			Content: theGroup.ReplicaInfo[message.FromServer].UserList, OriGroupName: theGroup.GroupName, LikeNumber: 0,
			FromServer: message.FromServer, Timestamp: theGroup.ReplicaInfo[message.FromServer].GroupNameTimestamp,
			GroupTimestamp: theGroup.MaxLamport}
		sm.SendingBetweenReplica(sString, replyStream, updateGroup)
	}

	startingM := message.Timestamp
	fmt.Printf("CheckForMerge(): for group: %v, local msg count of server %v is: %v\n",
		message.OriGroupName, message.FromServer, theGroup.ReplicaInfo[message.FromServer].MsgTimestamp)
	fmt.Printf("receive msg timestamp: %v\n", startingM)
	for theGroup.ReplicaInfo[message.FromServer].MsgTimestamp > startingM {
		println("CheckForMerge(): send M: ", startingM+1)
		dataMassage := theGroup.ReplicaInfo[message.FromServer].MessageLamport[startingM+1].Message
		println("content", dataMassage.Content)
		println("creator", dataMassage.Creator)
		println("lamport", dataMassage.Lamport)
		println("Replica", dataMassage.Replica)
		println("TotalLike", dataMassage.TotalLike)
		println(dataMassage.TotalLike)
		SendMessage := &ClientRequest{Request: "UpdateMsg", UserName: dataMassage.Creator,
			Content: dataMassage.Content, OriGroupName: theGroup.GroupName, LikeNumber: dataMassage.TotalLike,
			Replica: dataMassage.Replica, Timestamp: dataMassage.Lamport, FromServer: message.FromServer,
			MapIndex: startingM + 1}
		sm.SendingBetweenReplica(sString, replyStream, SendMessage)
		startingM++
	}
	startingL := message.LikeNumber
	fmt.Printf("CheckForMerge(): for group: %v, local like count of server %v is: %v\n",
		message.OriGroupName, message.FromServer, theGroup.ReplicaInfo[message.FromServer].LikeTimestamp)
	fmt.Printf("receive msg timestamp: %v\n", startingL)
	for theGroup.ReplicaInfo[message.FromServer].LikeTimestamp > startingL {
		println("CheckForMerge(): send L: ", startingL+1)

		dataNode := theGroup.ReplicaInfo[message.FromServer].LikeEventLamport[startingL+1]
		println(dataNode.Activity.UserName)
		println(dataNode.Activity.ActualTime.Format(time.RFC3339))
		println(dataNode.Activity.Like)
		println(dataNode.Activity.MsgReplica)
		println(dataNode.Activity.MsgLamport)
		println(message.FromServer)
		SendLike := &ClientRequest{Request: "UpdateLike", UserName: dataNode.Activity.UserName,
			Content: dataNode.Activity.ActualTime.Format(time.RFC3339), OriGroupName: theGroup.GroupName,
			LikeNumber: dataNode.Activity.Like, Replica: dataNode.Activity.MsgReplica,
			Timestamp: dataNode.Activity.MsgLamport, FromServer: message.FromServer, MapIndex: startingL + 1}
		sm.SendingBetweenReplica(sString, replyStream, SendLike)
		startingL++
	}
	println("CheckForMerge(): done")
	// group is not empty
}

func (sm *Server) ReplicaBroadcast(message *ClientRequest) {
	// iterate over the map StreamSet
	for server, config := range sm.StreamSet {
		// send the message to each stream
		sm.SendingBetweenReplica(server, config.Stream, message)
	}
}

func (sm *Server) SendingBetweenReplica(server string, stream Chat_ChatRoomClient, message *ClientRequest) {
	err := stream.Send(message)
	println("Sending ClientBroadcast message to server: " + server)
	if err != nil {
		//log.Fatalf("Error when sending message: %v", err)
		fmt.Printf("Error when sending message: %v\n", err)
		println("Trying to reconnect to server: " + server)
		/****** Create client instance ******/
		Config := sm.StreamSet[server]
		newStream, errr := Config.Client.ChatRoom(Config.Context)
		if errr != nil {
			println("Still can't connect to server: " + server)
			sm.StreamSet[server].isOnline = 0
		} else {
			println("Reconnected to server: " + server)
			sm.StreamSet[server].isOnline = 1
			sm.StreamSet[server].Stream = newStream
			errrr := newStream.Send(message)
			if errrr != nil {
				println("SendingBetweenReplica(): Error when sending message: " + errrr.Error())
			}
		}
	}
	/*else {
		sm.StreamSet[server].isOnline = 1
	} */
	println("SendingBetweenReplica(): done")
}

// AntiEntropy periodically sends the history of this server to all other servers
func (sm *Server) AntiEntropy() {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	// Send the history of this server to all other servers
	for _, group := range sm.GroupMap {
		println("IterateHistory(): sending history of group " + group.GroupName)
		sm.SendGroupInfo(group)
		sm.SendMessageList(group)
	}
	return
}

// SendGroupInfo sends the updated group info to every replica
func (sm *Server) SendGroupInfo(group *Group) {
	fmt.Printf("SendGroupInfo(): sending group: %v's info to all replicas\n", group.GroupName)
	for replica, info := range group.ReplicaInfo {
		if info.UserList != "" {
			fmt.Printf("SendGroupInfo(): Ready to broadcasting server %v 's participants list about group %v\n",
				replica, group.GroupName)
			i, _ := strconv.ParseInt(replica, 10, 32)
			fmt.Printf("i = %v\n", i)
			update := &ClientRequest{Request: "SendGroupInfo", Content: info.UserList,
				Replica: int32(i), Timestamp: info.GroupNameTimestamp, OriGroupName: group.GroupName}
			sm.ReplicaBroadcast(update)
		}
	}
	println("SendGroupInfo(): successfully sent group info to all replicas")
}

func (sm *Server) SendMessageList(theGroup *Group) {
	cur := theGroup.MessageList.Head
	for cur != nil {
		test := &ClientRequest{Request: "UpdateMsg", OriGroupName: theGroup.GroupName,
			Content: cur.Message.Content, UserName: cur.Message.Creator,
			Timestamp: cur.Message.Lamport, Replica: cur.Message.Replica, LikeNumber: cur.Message.TotalLike}
		sm.ReplicaBroadcast(test)
		sm.SendLikeList(cur.Message, theGroup)
		cur = cur.Next
	}
	println("SendMessageList() done on group: ", theGroup.GroupName)
	println("")
}

func (sm *Server) SendLikeList(theMessage *Message, theGroup *Group) {
	cur := theMessage.ActivityList.Head
	for cur != nil {
		timeString := cur.Activity.ActualTime.Format(time.RFC3339)
		test := &ClientRequest{Request: "UpdateLike", OriGroupName: theGroup.GroupName,
			Content: timeString, UserName: cur.Activity.UserName,
			Timestamp: theMessage.Lamport, Replica: theMessage.Replica, LikeNumber: cur.Activity.Like}
		sm.ReplicaBroadcast(test)
		cur = cur.Next
	}
	println("SendLikeList() done on Msg ID: ", theMessage.MsgId)
	println("")
}

// ReplicaSetUp sets up the connection with each replica
func (sm *Server) ReplicaSetUp(server string) {
	conn, err := grpc.Dial("172.30.100.10"+server+":"+"12000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("dialing:", err)
	}
	/****** Create client instance ******/
	client := NewChatClient(conn)
	ctx := context.Background()
	Stream, err := client.ChatRoom(ctx)
	for err != nil {
		Stream, err = client.ChatRoom(ctx)
		// sleep for 1 second
		time.Sleep(500 * time.Millisecond)
		//println("retrying to connect to server" + server)
	}
	sm.StreamSet[server] = &ServerConfig{Stream: Stream, Client: client, Context: ctx}
	println("ReplicaSetUp(): successfully connected to server" + server)
	errr := Stream.Send(&ClientRequest{Request: "ReplicaSetUp", UserName: sm.StringMe})
	if errr != nil {
		println("ReplicaSetUp(): Error when sending message: " + errr.Error())
	}
	println("")
}

// Merge :flag == 0 means insert message, flag == 1 means insert like, flag == 2 means insert group member
func (sm *Server) Merge(message *ClientRequest, flag int) *Group {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	_, flushGroup := sm.Insertion(message, flag, 1)
	// flush the screen for every connected clients' screens of this group
	println("Merge(): Done")
	return flushGroup
}

// Insertion inserts a message/like into the group or updates the group member list from the replica
func (sm *Server) Insertion(ClientMessage *ClientRequest, flag int, flag2 int) (*Node, *Group) {
	// unpack the ClientMessage
	if flag == 0 {
		fmt.Printf("Insertion(): Insert Msg: %v\n", ClientMessage.Content)
	} else if flag == 1 {
		fmt.Printf("Insertion(): Insert Like(time): %v\n", ClientMessage.Content)
	} else {
		fmt.Printf("Insertion(): Insert Group Member: %v, timeStamp: %v\n", ClientMessage.Content,
			ClientMessage.Timestamp)
	}
	the_group := sm.GroupMap[ClientMessage.OriGroupName]
	if the_group == nil {
		the_group = sm.CreateGroup(ClientMessage.OriGroupName, ClientMessage.GroupTimestamp)
		sm.GroupMap[ClientMessage.OriGroupName] = the_group
		if flag == 2 {
			println("Merge(): Successfully Update group member list for replica: ", ClientMessage.FromServer)
			the_group.ReplicaInfo[ClientMessage.FromServer].UserList = ClientMessage.Content
			the_group.ReplicaInfo[ClientMessage.FromServer].GroupNameTimestamp = ClientMessage.Timestamp
			return nil, the_group
		}

		messageCreated := newMessage(ClientMessage.Timestamp, ClientMessage.Replica, ClientMessage.UserName,
			ClientMessage.Content, 0)
		newNode := the_group.MessageList.Append()
		if flag2 == 1 {
			the_group.ReplicaInfo[ClientMessage.FromServer].MessageLamport[ClientMessage.MapIndex] = newNode
			fmt.Printf("Merge(): mapindex: %v user: %v content: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.Content)
			if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp+1 {
				the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp++
				println("Merge(): increment MsgTimestamp ack by 1")
			}
		}

		newNode.Message.MsgId = the_group.MessageList.Count
		newNode.Message = messageCreated
		the_group.MessageList.TenStart = newNode
		println("Merge(): New ClientMessage created")
		if flag == 1 { // like
			println("Merge(): 172")
			Activity := sm.InsertLike(newNode, ClientMessage)
			Activity.Activity.MsgLamport = ClientMessage.Timestamp
			Activity.Activity.MsgReplica = ClientMessage.Replica
			the_group.ReplicaInfo[ClientMessage.FromServer].LikeEventLamport[ClientMessage.MapIndex] = Activity
			fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.Content)

			if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp+1 {
				the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp++
				println("Merge(): increment like Timestamp ack by 1")
			}
			newNode.Message.TotalLike = sm.CalculateLike(newNode)
		}
	} else {

		if flag == 0 {
			if flag2 == 1 {
				if the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp >= ClientMessage.MapIndex {
					println("Redundant message")
					return nil, nil
				}
			}
			// flag = 0 means append the ClientMessage
			// find the place where the ClientMessage should be inserted into the group
			println("Merge(): found group to merge")
			result, do := sm.LamportPlace(ClientMessage, the_group)
			if result != nil {
				fmt.Printf("Merge(): Insertion message after Lamport: %v, Replica:\n",
					result.Message.Lamport, result.Message.Replica)
				temp := result.Next
				messageCreated := newMessage(ClientMessage.Timestamp, ClientMessage.Replica, ClientMessage.UserName,
					ClientMessage.Content, ClientMessage.LikeNumber)
				newNode := &Node{Message: messageCreated, Next: nil}
				result.Next = newNode
				newNode.Next = temp
				sm.UpdateGroupLamport(the_group, ClientMessage.Timestamp)
				the_group.MessageList.Count++
				the_group.MessageList.UpdateTenStart(newNode)
				the_group.MessageList.IndexReorder()
				if flag2 == 1 {
					the_group.ReplicaInfo[ClientMessage.FromServer].MessageLamport[ClientMessage.MapIndex] = newNode
					fmt.Printf("Merge(): mapindex: %v user: %v content: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.Content)
					if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp+1 {
						the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp++
						println("Merge(): increment MsgTimestamp ack by 1")
					}
				}
				if do == false {
					// append to the tail
					println("Merge(): Append Message as the tail")
					the_group.MessageList.Tail = newNode
				}
				return newNode, the_group
			} else {
				if do == true {
					println("Merge(): Insert Message as the head")
					// append as the head
					messageCreated := newMessage(ClientMessage.Timestamp, ClientMessage.Replica, ClientMessage.UserName,
						ClientMessage.Content, ClientMessage.LikeNumber)
					newNode := &Node{Message: messageCreated, Next: nil}
					newNode.Next = the_group.MessageList.Head
					the_group.MessageList.Count++
					if the_group.MessageList.Head == nil {
						the_group.MessageList.Tail = newNode
					}
					the_group.MessageList.Head = newNode
					the_group.MessageList.UpdateTenStart(newNode)
					sm.UpdateGroupLamport(the_group, ClientMessage.Timestamp)
					the_group.MessageList.IndexReorder()
					if flag2 == 1 {
						the_group.ReplicaInfo[ClientMessage.FromServer].MessageLamport[ClientMessage.MapIndex] = newNode
						fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.Content)
						if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp+1 {
							the_group.ReplicaInfo[ClientMessage.FromServer].MsgTimestamp++
							println("Merge(): increment MsgTimestamp ack by 1")
						}
					}
					return newNode, the_group
				}
				println("Merge(): Insert Message failed, just replace the information")
				return nil, nil
			}
		} else if flag == 1 { // check the like
			if the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp >= ClientMessage.MapIndex {
				println("Redundant like")
				return nil, nil
			}
			MsgToLike := the_group.MessageList.Head
			if MsgToLike == nil {
				println("Merge(): the group does not have messages, insert a message first")
				println("Merge(): Then put a like event associated with the message")
				InsertedMsg, _ := sm.Insertion(ClientMessage, 0, 0)
				ActivityNode := sm.InsertLike(InsertedMsg, ClientMessage)
				ActivityNode.Activity.MsgLamport = ClientMessage.Timestamp
				ActivityNode.Activity.MsgReplica = ClientMessage.Replica
				the_group.ReplicaInfo[ClientMessage.FromServer].LikeEventLamport[ClientMessage.MapIndex] = ActivityNode
				if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp+1 {
					the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp++
				}
				fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.LikeNumber)
				InsertedMsg.Message.TotalLike = sm.CalculateLike(InsertedMsg)
				return ActivityNode, nil
			}
			for MsgToLike != nil {
				if MsgToLike.Message.Lamport == ClientMessage.Timestamp && MsgToLike.Message.Replica == ClientMessage.Replica {
					println("Merge(): the ClientMessage of like update is found")
					ActivityNode := sm.InsertLike(MsgToLike, ClientMessage)
					ActivityNode.Activity.MsgLamport = ClientMessage.Timestamp
					ActivityNode.Activity.MsgReplica = ClientMessage.Replica

					previousLike := MsgToLike.Message.TotalLike
					MsgToLike.Message.TotalLike = sm.CalculateLike(MsgToLike)
					the_group.ReplicaInfo[ClientMessage.FromServer].LikeEventLamport[ClientMessage.MapIndex] = ActivityNode
					fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.LikeNumber)
					if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp+1 {
						the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp++
						println("Merge(): increment LikeTimestamp ack by 1")
					}
					var groupPtr *Group
					if previousLike != MsgToLike.Message.TotalLike {
						groupPtr = the_group
					} else {
						groupPtr = nil
					}
					return ActivityNode, groupPtr
				} else {
					MsgToLike = MsgToLike.Next
				}
			}
			println("Merge(): the Message to like is not found")
			// create a new ClientMessage and append it to the group
			InsertedMsg, _ := sm.Insertion(ClientMessage, 0, 0)
			ActivityNode := sm.InsertLike(InsertedMsg, ClientMessage)
			ActivityNode.Activity.MsgLamport = ClientMessage.Timestamp
			ActivityNode.Activity.MsgReplica = ClientMessage.Replica
			the_group.ReplicaInfo[ClientMessage.FromServer].LikeEventLamport[ClientMessage.MapIndex] = ActivityNode
			fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.LikeNumber)
			if ClientMessage.MapIndex == the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp+1 {
				the_group.ReplicaInfo[ClientMessage.FromServer].LikeTimestamp++
				println("Merge(): increment LikeTimestamp ack by 1")
			}
			fmt.Printf("Merge(): mapindex: %v user: %v like: %v\n", ClientMessage.MapIndex, ClientMessage.UserName, ClientMessage.LikeNumber)
			InsertedMsg.Message.TotalLike = sm.CalculateLike(InsertedMsg)
			return ActivityNode, nil
		} else if flag == 2 {
			println("Merge(): 252", ClientMessage.Content)
			if ClientMessage.Timestamp > the_group.ReplicaInfo[ClientMessage.FromServer].GroupNameTimestamp {
				println("Merge():Successfully Update group member list for replica: ", ClientMessage.Replica)
				the_group.ReplicaInfo[ClientMessage.FromServer].UserList = ClientMessage.Content
				the_group.ReplicaInfo[ClientMessage.FromServer].GroupNameTimestamp = ClientMessage.Timestamp
				return nil, the_group
			}
			println("Merge(): Insert Group Member list is out of date")
			return nil, nil
		}
	}
	return nil, nil
}

// InsertLike inserts a like activity to the ClientMessage
func (sm *Server) InsertLike(MsgToLike *Node, message *ClientRequest) *Node {
	println("InsertLike(): searching place to put event " + message.Content)
	ActualTime, _ := time.Parse(time.RFC3339, message.Content)
	if MsgToLike.Message.ActivityList == nil {
		// list is null
		println("InsertLike(): list is null, insert as head: " + message.Content)
		MsgToLike.Message.ActivityList = &List{Head: nil, TenStart: nil, Count: 0}
		newNode := MsgToLike.Message.ActivityList.Append()
		newNode.Activity.Like = message.LikeNumber
		newNode.Activity.UserName = message.UserName
		return newNode
	}

	cur := MsgToLike.Message.ActivityList.Head
	println("InsertLike(): 285")
	if cur == nil || cur.Activity.ActualTime.After(ActualTime) {
		// insert at head
		println("InsertLike(): insert as head:" + message.Content)
		newNode := &Node{Activity: &Activity{UserName: message.UserName, ActualTime: ActualTime,
			Like: message.LikeNumber}, Next: nil}
		MsgToLike.Message.ActivityList.Head = newNode
		newNode.Next = cur
		return newNode
	}
	println("295")
	for cur != nil {
		fmt.Printf("LinkedList: user:%v like:%v \n", cur.Activity.UserName, cur.Activity.Like)
		if cur.Activity.ActualTime.Before(ActualTime) {
			// insert before cur
			println("InsertLike(): insert before cur")
			if cur.Next == nil {
				fmt.Printf("InsertLike(): insert %v after User %v's like record)\n",
					message.Content, cur.Activity.UserName)
				temp := cur.Next
				newNode := &Node{Activity: &Activity{UserName: message.UserName, ActualTime: ActualTime,
					Like: message.LikeNumber}, Next: nil}
				cur.Next = newNode
				newNode.Next = temp
				return newNode
			} else {
				println("InsertLike(): 310")
				if cur.Next.Activity.ActualTime.After(ActualTime) {
					fmt.Printf("InsertLike(): insert %v after User %v's like record)\n",
						message.Content, cur.Next.Activity.UserName)
					temp := cur.Next
					newNode := &Node{Activity: &Activity{UserName: message.UserName, ActualTime: ActualTime,
						Like: message.LikeNumber}, Next: nil}
					cur.Next = newNode
					newNode.Next = temp
					return newNode
				} else {
					cur = cur.Next
				}
			}
		} else {
			cur = cur.Next
		}
	}
	println("InsertLike(): insert failed")
	return nil
}

// CalculateLike calculates the total like number of a message
func (sm *Server) CalculateLike(Msg *Node) int32 {
	cur := Msg.Message.ActivityList.Head
	var sum int32
	sum = 0
	fmt.Printf("CalculatingLike(): MsgID: %v\n", Msg.Message.MsgId)
	Msg.Message.UserLike = make(map[string]int32)
	for cur != nil {
		like, ok := Msg.Message.UserLike[cur.Activity.UserName]
		if ok {
			// Found the user in the map
			if like == cur.Activity.Like {
				println("CalculateLike(): Illegal like behavior detected: " + cur.Activity.UserName + "repeated request")
			} else {
				Msg.Message.UserLike[cur.Activity.UserName] = cur.Activity.Like
				sum += cur.Activity.Like
			}
		} else {
			// can't find the user in the map
			if cur.Activity.Like == 1 {
				Msg.Message.UserLike[cur.Activity.UserName] = 1
				sum++
			} else {
				println("CalculateLike(): Illegal dislike behavior detected: " + cur.Activity.UserName + "haven't liked the message")
			}
		}
		cur = cur.Next
	}
	println("CalculatingLike(): Done")
	return sum
}

// LamportPlace finds the correct place to insert the message based on lamport timestamp

// UpdateGroupLamport updates the group's lamport timestamp if the new lamport is larger
func (sm *Server) UpdateGroupLamport(group *Group, Lamport int32) {
	if Lamport > group.MaxLamport {
		group.MaxLamport = Lamport
		println("UpdateGroupLamport(): Update the group's lamport")
	}
}
