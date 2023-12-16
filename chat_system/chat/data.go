package chat

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type List struct {
	Head     *Node
	Tail     *Node
	TenStart *Node
	Count    int32
}

type Node struct {
	Activity *Activity
	Message  *Message
	Next     *Node
}

type Activity struct {
	UserName   string
	ActualTime time.Time
	Like       int32
	MsgLamport int32
	MsgReplica int32
}

type Message struct {
	MsgId        int32            `json:"msg_id"`
	Lamport      int32            `json:"lamport"`  // lamport timestamp
	Replica      int32            `json:"replica"`  // which replica does this message come from
	Creator      string           `json:"creator"`  // the username of the creator
	Content      string           `json:"content"`  // the content of the message
	UserLike     map[string]int32 `json:"userLike"` // 1 indicates like, -1 indicates dislike
	ActivityList *List            `json:"-"`
	TotalLike    int32            `json:"totalLike"`
}

type Server struct {
	UnimplementedChatServer
	GroupMap  map[string]*Group
	TotalUser *List
	Mu        sync.Mutex
	Me        int32
	StringMe  string
	UserCount int32
	StreamSet map[string]*ServerConfig
}

type ServerConfig struct {
	Client   ChatClient
	Stream   Chat_ChatRoomClient
	Context  context.Context
	LastTime time.Time
	isOnline int32
}

type Group struct {
	GroupName         string
	MessageList       *List
	UserList          map[int32]*User  // track the connecting clients to this group
	sameUserNameCount map[string]int32 // track the identities (UserNames) of this group
	MaxLamport        int32
	ReplicaInfo       map[string]*Information
}

type User struct {
	UserName string
	UserId   int32
	context  Chat_ChatRoomServer
	Next     *User
}

type Information struct {
	UserList           string
	GroupNameTimestamp int32 // last received message's timestamp from this server
	LikeTimestamp      int32
	MsgTimestamp       int32
	LikeEventLamport   map[int32]*Node
	MessageLamport     map[int32]*Node
}

/*
	for each group:

create a Message file for the group Message
create a User file for the group User
create a LikeActivity file for the group LikeActivity

for each message in the group:

	append the message to the Message file
	for each like activity in the message:
		append the like activity to the LikeActivity file

for each user in the group:

	append the user to the User file
*/

func newMessage(Lamport int32, Replica int32, Creator string, Content string, like int32) *Message {
	return &Message{0, Lamport, Replica, Creator,
		Content, make(map[string]int32), &List{nil, nil, nil, 0}, like}
}

func (l *List) Append() *Node {
	newNode := &Node{&Activity{}, &Message{}, nil}
	if l.Tail != nil {
		l.Tail.Next = newNode
		l.Tail = newNode
	}
	if l.Head == nil {
		l.Head = newNode
		l.Tail = newNode
	}
	l.Count += 1
	return newNode
}

/*
	func (l *List) FindLamport(lamport int32) (*Node, bool) {
		head := l.Head
		dummy_head := &Node{Next: nil}
		dummy_head.Next = head
		cur := dummy_head
		for cur.Next != nil {
			if cur.Next.Message.Lamport == lamport {
				return cur, false
			} else {
				cur = cur.Next
			}
		}
		return nil, true
	}
*/
func (l *List) IndexReorder() {
	cur := l.Head
	prev := &Node{Message: &Message{MsgId: 0}, Next: nil}
	for cur != nil {
		if cur.Message.MsgId != prev.Message.MsgId+1 {
			// reorder the index
			cur.Message.MsgId = prev.Message.MsgId + 1
		}
		prev = cur
		cur = cur.Next
	}
}

func (l *List) UpdateTenStart(insert *Node) {
	if l.Count <= 10 {
		l.TenStart = l.Head
		return
	}
	if l.TenStart.Next == insert {
		l.TenStart = insert
		return
	}
}

func (l *List) Delete(id int32) *Node {
	head := l.Head
	dummy_head := &Node{}
	dummy_head.Next = head
	cur := dummy_head
	var return_node *Node
	for cur.Next != nil {
		if cur.Next.Message.MsgId == id {
			if cur.Next == l.Tail {
				l.Tail = cur
			}
			return_node = cur.Next
			cur.Next = cur.Next.Next
		} else {
			cur = cur.Next
		}
	}
	l.Head = dummy_head.Next
	return return_node
}

func (l *List) Display() {
	println("display the list")
	head := l.Head
	for head != nil {
		fmt.Printf("%v : %v,\n", head.Activity.UserName, head.Activity.Like)
		head = head.Next
	}
	println("")
}

/*
func (l *List) FindGroup(name string) *Node {
	head := l.Head
	for head != nil {
		if head.ObjectName == name {
			return head
		}
		head = head.Next
	}
	return nil
}

func (l *List) FindUser(id int32) *Node {
	head := l.Head
	for head != nil {
		if head.ObjectId == id {
			return head
		}
		head = head.Next
	}
	return nil
}
*/
