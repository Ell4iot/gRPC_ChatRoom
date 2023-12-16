package chat

import (
	"log"
	"strconv"
	"time"
)

func (sm *Server) test(message *ClientRequest) {
	//group := sm.testPrepare()
	sm.Merge(message, 0)
	//group.MessageList.Display()
	//println("group.MessageList.Tail.ObjectName: ", group.MessageList.Tail.ObjectName)
	//group.MessageList.IndexReorder()
	//group.MessageList.Display()
}

func (sm *Server) testPrepare() *Node {
	/*
		the_group := sm.GroupList.Append("test", 0)
		the_group.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		n1 := the_group.MessageList.Append("testA", 1)
		n1.Content = "AAAAAAAAAAAAA"
		n1.LamportTs = 1
		n1.ReplicaNo = 1
		n1.Like = 0
		n1.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		UserLike := make(map[string]int32)
		n1.UserLike = UserLike
		n2 := the_group.MessageList.Append("testB", 2)
		n2.LamportTs = 2
		n2.ReplicaNo = 1
		n2.Like = 0
		n2.Content = "BBBBBBBBBBBB"
		UserLike2 := make(map[string]int32)
		n2.UserLike = UserLike2
		n2.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		n3 := the_group.MessageList.Append("testB2", 3)
		n3.LamportTs = 2
		n3.ReplicaNo = 2
		n3.Like = 0
		n3.Content = "B2222222222222"
		n3.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		UserLike3 := make(map[string]int32)
		n3.UserLike = UserLike3
		n4 := the_group.MessageList.Append("testD", 4)
		n4.LamportTs = 2
		n4.ReplicaNo = 4
		n4.Like = 0
		n4.Content = "DDDDDDDDDDDDD"
		UserLike4 := make(map[string]int32)
		n4.UserLike = UserLike4
		n4.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		n5 := the_group.MessageList.Append("testE", 5)
		n5.LamportTs = 2
		n5.ReplicaNo = 5
		n5.Like = 0
		n5.Content = "EEEEEEEEEEEEE"
		n5.MessageList = &List{Head: nil, TenStart: nil, Count: 0}
		UserLike5 := make(map[string]int32)
		n5.UserLike = UserLike5
		return the_group
	*/
	return nil
}

func (sm *Server) testLike(message *ClientRequest) {
	//sm.likePrepare()
	sm.testPrepare()
	// 1 insert a like message
	sm.Merge(message, 1)
	sm.PrintData()
}

func (sm *Server) Trying() {
	var i int32
	for i = 1; i <= 5; i++ {
		if i != sm.Me {
			testing := &ClientRequest{Request: "docker", Content: "Receiving from server " + strconv.Itoa(int(sm.Me))}
			Stream, ok := sm.StreamSet[strconv.Itoa(int(i))]
			for ok != true {
				Stream, ok = sm.StreamSet[strconv.Itoa(int(i))]
				time.Sleep(500 * time.Millisecond)
				println("waiting for server " + strconv.Itoa(int(i)) + " to connect")
			}
			println("Sending to server " + strconv.Itoa(int(i)))
			err := Stream.Stream.Send(testing)
			if err != nil {
				log.Fatalf("Error when sending message: %v", err)
			}
		}
	}
}

func (sm *Server) testMerge() {
	g := sm.CreateGroup("test", 0)
	g.ReplicaInfo["1"] = &Information{UserList: ", s1, s1, s1, s1, "}
	g.ReplicaInfo["2"] = &Information{UserList: ", s2, s2, s2, s2, "}
	g.ReplicaInfo["3"] = &Information{UserList: ", s3, s3, s3, s3, "}
	g.ReplicaInfo["4"] = &Information{UserList: ", s4, s4, s4, s4, "}
	g.ReplicaInfo["5"] = &Information{UserList: ", s5, s5, s5, s5, "}
	s := NameMerge(g)
	println(s)
}
