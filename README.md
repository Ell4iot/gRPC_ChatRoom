# CS 2510 Project 2

Group member:  Yuhang Wang, Yuxuan Zhang

Contact: 

Email:  YUW199@pitt.edu, YUZ276@pitt.edu,

## Docker Commands

We strictly follow the instruction from the prompt

## File Composition

Once you enter the container, you should be able to see the following files:

1. A folder named `chat`
2. `client.go` : the client file
3. `go.mod`
4. `go.sum`
5. `server.go`: the server file

In `/chat` you should be able to see the following files:

1. `chat.pb.go`
2. `chat.proto` : the proto file we used to configure our GRPC
3. `chat_grpc.pb.go`
4. `chatserver.go`: utility functions for the server side 
5. `data.go`: data structures used on the server side
6. `replica.go`:  functions for insert messages after partition healed, sending between replicas, anti-entropy functions.
7. `HelperFunctions.go`:helper functions

## Run Our System

- We strictly follow the instruction prompt.

- Please notice that for client command, please do not enter extra white space before the whole command:
  - For example, `space` u `space` Username, will not be accepted
  - Please enter: u `space` UserName
  - It is the same among all other commands.
- When testing our program, I found that the Internet may delay the message, so if after the partition is healed and the screen does not refresh on time, please patiently wait for it to refresh. Or, you can simply type `222` on the client program, to force the connecting server to do the `anti-entropy` message merge function.

## System Instructions

We would like to mention that the command on the client side should be entered seperately in our system. To be more specific, we take login command`u Amy` as an example:

* You should first enter `u` and then press enter.
* Then you should be able to see further instructions telling you "Please login using your username:".
* Now you should type in `Amy` to login.
* ==All functions and features are tested and work properly.==
* The client program ==automatically connects== to the server without inputing `c localhost:12000`

It is the same for all of our commands, you should type the function you want to use and wait for further instructions and enter required information to proceed.
