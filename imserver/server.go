package imserver

import (
	"errors"
	"fmt"
	"imsystem/pack"
	"imsystem/pb"
	"net"
	"strconv"
	"sync"
)

type ServerInterface interface {
	UserOffline(*User)
	Broadcast(*pb.BroadcastPush)
	Query() []string
	Rename(*User, string) bool
	PrivateChat(*User, string, string) error
}

type Server struct {
	Ip            string
	Port          int
	onlineUsers   map[string]*User
	userLock      sync.RWMutex
	broadcastChan chan []byte
	packer        *pack.Packer
}

func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:            ip,
		Port:          port,
		onlineUsers:   make(map[string]*User),
		broadcastChan: make(chan []byte),
		packer:        pack.NewPacker(),
	}
}

func (s *Server) Start() {
	//listen
	listener, err := net.Listen("tcp", net.JoinHostPort(s.Ip, strconv.Itoa(s.Port)))
	if err != nil {
		fmt.Println("listen error:", err)
	}
	//defer close
	defer listener.Close()
	//listen broadcast
	go s.listenBroadcast()
	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		//requestHandler
		go s.handler(conn)
	}
}

func (s *Server) Broadcast(body *pb.BroadcastPush) {
	bytes, err := s.packer.Encode(pack.NewPushHead(pb.PushType_PUSH_TYPE_BROADCAST), body)
	if err != nil {
		fmt.Println("broadcast error:", err)
	}
	s.broadcastChan <- bytes
}

func (s *Server) UserOffline(user *User) {
	s.userLock.Lock()
	delete(s.onlineUsers, user.Name)
	s.userLock.Unlock()

	//下线广播
	s.Broadcast(&pb.BroadcastPush{
		User:    user.String(),
		Content: "下线",
	})
}

func (s *Server) Query() []string {
	userNames := make([]string, 0, len(s.onlineUsers))
	s.userLock.RLock()
	defer s.userLock.RUnlock()
	for _, user := range s.onlineUsers {
		userNames = append(userNames, user.String())
	}
	return userNames
}

func (s *Server) Rename(user *User, newName string) bool {
	s.userLock.Lock()
	defer s.userLock.Unlock()
	_, ok := s.onlineUsers[newName]
	if ok {
		return false
	}
	delete(s.onlineUsers, user.Name)
	s.onlineUsers[newName] = user
	user.Name = newName
	return true
}

func (s *Server) PrivateChat(user *User, toUserName string, content string) error {
	s.userLock.RLock()
	defer s.userLock.RUnlock()
	toUser, ok := s.onlineUsers[toUserName]
	if !ok {
		return errors.New("查无此人:" + toUserName)
	}
	bytes, err := s.packer.Encode(pack.NewPushHead(pb.PushType_PUSH_TYPE_PRIVATE_CHAT), &pb.PrivateChatPush{
		User:    user.String(),
		Content: content,
	})
	if err != nil {
		return err
	}
	toUser.SendMessage(bytes)
	return nil
}

func (s *Server) userOnline(user *User) {
	s.userLock.Lock()
	s.onlineUsers[user.Name] = user
	s.userLock.Unlock()

	user.Online()
	//上线广播
	s.Broadcast(&pb.BroadcastPush{
		User:    user.String(),
		Content: "上线",
	})
}

func (s *Server) handler(conn net.Conn) {
	user := NewUser(conn, s)
	s.userOnline(user)
}

func (s *Server) listenBroadcast() {
	for {
		select {
		case message := <-s.broadcastChan:
			s.userLock.RLock()
			for _, user := range s.onlineUsers {
				user.WriteMessage(message)
			}
			s.userLock.RUnlock()
		}
	}
}
