package chat

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// CreateGroup creates new group and append it to the total group list
func (sm *Server) CreateGroup(GroupName string, Timestamp int32) *Group {
	the_group := &Group{GroupName: GroupName, MaxLamport: Timestamp}
	the_group.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
	the_group.ReplicaInfo = make(map[string]*Information)
	the_group.UserList = make(map[int32]*User)
	the_group.sameUserNameCount = make(map[string]int32)
	the_group.ReplicaInfo[sm.StringMe] = &Information{UserList: "",
		MsgTimestamp: 0, GroupNameTimestamp: 0, LikeTimestamp: 0, LikeEventLamport: make(map[int32]*Node), MessageLamport: make(map[int32]*Node)}

	for i := int32(1); i <= 5; i++ {
		if i != sm.Me {
			s := strconv.Itoa(int(i))
			the_group.ReplicaInfo[s] = &Information{UserList: "",
				MsgTimestamp: 0, LikeTimestamp: 0, GroupNameTimestamp: 0, LikeEventLamport: make(map[int32]*Node), MessageLamport: make(map[int32]*Node)}

		}
	}
	fmt.Printf("new group created: %v\n", GroupName)
	println("CreateGroup() Done\n")
	file, err := os.Open("group.txt")
	if err != nil {
		println("CreateGroup(): open err\n")
	} else {
		writer := bufio.NewWriter(file)
		_, _ = writer.WriteString(GroupName + "\n")
		writer.Flush()
		file.Close()
	}
	return the_group
}

func (sm *Server) FlushScreen(group *Group) {
	println("FlushScreen()")
	for _, client := range group.UserList {
		sm.ShowHistory(group.GroupName, &client.context)
	}
	println("FlushScreen() for all clients in: ", group.GroupName)
}

func NameMerge(Group *Group) string {
	println("NameMerge()")
	returnString := ", "
	var i int32
	for i = 1; i <= 5; i++ {
		Names := strings.Fields(Group.ReplicaInfo[strconv.Itoa(int(i))].UserList)
		for _, name := range Names {
			if strings.Contains(returnString, ", "+name+", ") == false {
				returnString = returnString + name + ", "
			}
		}
	}
	returnString = returnString[2 : len(returnString)-2]
	println("NameMerge() Done\n")
	return returnString
}

func (sm *Server) PerformingLike(group *Group, like int32, line *Message, UserName string) (string, int32) {
	if like == 1 {
		println("Increasing like for message: ", line.MsgId)
		line.TotalLike++
	} else {
		println("Decreasing like for message: ", line.MsgId)
		line.TotalLike--
	}
	line.UserLike[UserName] = like
	Now := time.Now()
	activity := line.ActivityList.Append()
	activity.Activity.UserName = UserName
	activity.Activity.ActualTime = Now
	activity.Activity.Like = like
	activity.Activity.MsgReplica = line.Replica
	activity.Activity.MsgLamport = line.Lamport
	group.ReplicaInfo[sm.StringMe].LikeTimestamp++
	index := group.ReplicaInfo[sm.StringMe].LikeTimestamp
	group.ReplicaInfo[sm.StringMe].LikeEventLamport[group.ReplicaInfo[sm.StringMe].LikeTimestamp] = activity
	println("PerformingLike() Done\n")
	return Now.Format(time.RFC3339), index
}

func (sm *Server) LamportPlace(message *ClientRequest, the_group *Group) (*Node, bool) {
	if the_group.MessageList.Head == nil {
		// insert as the head
		println("LamportPlace(): the group is empty")
		return nil, true
	}
	if the_group.MessageList.Head.Message.Lamport > message.Timestamp {
		// insert at head
		println("LamportPlace(): insert at head")
		return nil, true
	} else if the_group.MessageList.Head.Message.Lamport == message.Timestamp {
		if message.Replica < the_group.MessageList.Head.Message.Replica {
			println("LamportPlace(): insert at head")
			return nil, true
		} else if message.Replica == the_group.MessageList.Head.Message.Replica {
			fmt.Printf("Merge: message already exists: %v\n", message.OriGroupName)
			println("LamportPlace(): Replace the information")
			the_group.MessageList.Head.Message.Creator = message.UserName
			the_group.MessageList.Head.Message.Content = message.Content
			the_group.MessageList.Head.Message.TotalLike = message.LikeNumber
			return nil, false
		}
	} else if the_group.MessageList.Tail.Message.Lamport < message.Timestamp {
		return the_group.MessageList.Tail, false
	}

	// head.Lamport < message.Lamport < tail.Lamport
	curNode := the_group.MessageList.Head
	println("LamportPlace(): TimestampStart is found")
	for curNode.Next != nil {
		if curNode.Next.Message.Lamport == message.Timestamp {
			if curNode.Next.Message.Replica == message.Replica {
				fmt.Printf("Merge: message already exists: %v\n", message.OriGroupName)
				println("LamportPlace(): Replace the information")
				curNode.Next.Message.Creator = message.UserName
				curNode.Next.Message.Content = message.Content
				curNode.Next.Message.TotalLike = message.LikeNumber
				return nil, false
			} else if curNode.Next.Message.Replica > message.Replica {
				return curNode, true
			} else {
				// TimestampStart.ReplicaNo < message.Replica
				curNode = curNode.Next
			}
		} else if curNode.Next.Message.Lamport < message.Timestamp {
			curNode = curNode.Next
		} else if curNode.Next.Message.Lamport > message.Timestamp {
			return curNode, true
		}
	}
	println("LamportPlace(): reach the end of the list")
	return curNode, true
}

func (sm *Server) AddUsername(theGroup *Group, UserName string) {
	println("Add group username")
	number, ok := theGroup.sameUserNameCount[UserName]
	if ok {
		theGroup.sameUserNameCount[UserName] = number + 1
		fmt.Printf("Number of client with the same name %v: %v\n", UserName, theGroup.sameUserNameCount[UserName])
	} else {
		println("First user with this name")
		theGroup.sameUserNameCount[UserName] = 1
	}
	if theGroup.sameUserNameCount[UserName] == 1 {
		theGroup.ReplicaInfo[sm.StringMe].UserList = theGroup.ReplicaInfo[sm.StringMe].UserList +
			" " + UserName + " "
		theGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp++
		fmt.Printf("time stamp = %v for server: %v on group: %v\n",
			theGroup.ReplicaInfo[sm.StringMe].GroupNameTimestamp, sm.StringMe, theGroup.GroupName)
		println("New user list: ", theGroup.ReplicaInfo[sm.StringMe].UserList)
	}
	println("AddUsername() Done\n")
}

func (sm *Server) DeleteUsername(Group *Group, UserName string) {
	println("DeleteUsername(): check group username")
	Group.sameUserNameCount[UserName]--
	if Group.sameUserNameCount[UserName] == 0 {
		AllUserNames := Group.ReplicaInfo[sm.StringMe].UserList
		AllUserNames = strings.Replace(AllUserNames, " "+UserName+" ", " ", -1)
		Group.ReplicaInfo[sm.StringMe].UserList = AllUserNames
		Group.ReplicaInfo[sm.StringMe].GroupNameTimestamp++
		println("DeleteUsername() done\n")
		fmt.Printf("New user list: %v\n", Group.ReplicaInfo[sm.StringMe].UserList)
	}

}

func (sm *Server) PrintData() {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()
	println("Printing the data on server: ", sm.Me)
	for groupName, headGroup := range sm.GroupMap {
		fmt.Printf("Group: %v \n", groupName)
		fmt.Printf("Users list: \n")
		for _, client := range headGroup.UserList {
			println("Name: ", client.UserName)
		}
		println("")
		fmt.Printf("Messages list: %v messages\n", headGroup.MessageList.Count)
		msgNode := headGroup.MessageList.Head
		for msgNode != nil {
			println("-----------------------------")
			fmt.Printf("%vth msg, Lamport:%v Server:%v          ", msgNode.Message.MsgId,
				msgNode.Message.Lamport, msgNode.Message.Replica)
			fmt.Printf(msgNode.Message.Creator + ": " + msgNode.Message.Content)
			fmt.Printf("          like: %v\n", msgNode.Message.TotalLike)
			println("")
			println("showing activity:")
			ActivityNode := msgNode.Message.ActivityList.Head
			for ActivityNode != nil {
				HappenTime := ActivityNode.Activity.ActualTime.Format(time.RFC3339)
				fmt.Printf("Time: %v      User: %v  Like: %v\n", HappenTime, ActivityNode.Activity.UserName,
					ActivityNode.Activity.Like)
				ActivityNode = ActivityNode.Next
			}
			println("------------------------------")
			msgNode = msgNode.Next
		}
	}
	println("PrintData() done\n")
}

func (sm *Server) InspectMap() {

	for groupName, GroupNode := range sm.GroupMap {
		println("Group: ", groupName)
		fmt.Printf("Printing server: %v all messages: \n", sm.StringMe)
		for stS, tNode := range GroupNode.ReplicaInfo {
			fmt.Printf("%v Inspecting saved Server: %v\n", sm.StringMe, stS)
			var i int32
			for i = 1; i <= tNode.MsgTimestamp; i++ {
				fmt.Printf("Lamport: %v Replica %v Message: %v\n", tNode.MessageLamport[i].Message.Lamport,
					tNode.MessageLamport[i].Message.Replica, tNode.MessageLamport[i].Message.Content)
			}
			for i = 1; i <= GroupNode.ReplicaInfo[sm.StringMe].LikeTimestamp; i++ {
				fmt.Printf("Like Name: %v, like: %v， likeMessage Replica: %v, likeMessage Lamport: %v\n", tNode.LikeEventLamport[i].Activity.UserName,
					tNode.LikeEventLamport[i].Activity.Like, tNode.LikeEventLamport[i].Activity.MsgReplica,
					tNode.LikeEventLamport[i].Activity.MsgLamport)
			}
		}
	}
}

func (sm *Server) CheckGroup(Groupname string) {
	theGroup, ok := sm.GroupMap[Groupname]
	if !ok {
		// create new group
		theGroup = sm.CreateGroup(Groupname, 0)
		sm.GroupMap[Groupname] = theGroup
	}
}

func (sm *Server) AllGroup(ct Chat_ChatRoomServer) {
	for groupName := range sm.GroupMap {
		message := &ServerResponse{Request: "CheckGroup", Content: groupName, Username: sm.StringMe}
		err := ct.Send(message)
		if err != nil {
			println("AllGroup() error: ", err)
		}
	}
}

/*

c localhost:12000
u amy


*/
