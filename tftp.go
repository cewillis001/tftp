package tftp

import (
	"net"
	"errors"
	"fmt"
	"time"
)

type File struct {
	Name string
	Data []byte
	Conn *net.UDPConn
	Raddr *net.UDPAddr
}

func min(a int, b int) int{
	if(a <= b) { return a } else { return b }
}

func checkError(err error){
	if (err != nil) {
		fmt.Println("Error: ", err)
	}
}

func makeRRQ(filename string) ([]byte, error ){
	packet, err := makeWRQ(filename)
	if(packet != nil){ 
		packet[1] = 1
	}
	return packet, err
}

func SendRRQ(filename string, Conn *net.UDPConn){
	msg, err := makeRRQ(filename)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)
}

func SendRRQTo(filename string, Conn *net.UDPConn, raddr *net.UDPAddr){
  msg, err := makeRRQ(filename)
  checkError(err)
  _,err = Conn.WriteTo(msg, raddr)
  checkError(err)
}

func HandleRRQ(filename string, in <-chan []byte, done chan<- *net.UDPAddr, Conn *net.UDPConn, raddr *net.UDPAddr, f []byte) {
	index := 0
	max := min(512, len(f))
	data := f[index : max]
	block := []byte{0, 1}
	SendDATATo(block, data, Conn, raddr)
	for {
		select {
			case r := <- in:
				ackBlock, err := GetAckBlock(r)
				if(err != nil){
					//badly formed ack packet
					errCode := []byte{0,0}
					errMsg  := "badly formed ack packet"
					SendERRORTo(errCode, errMsg, Conn, raddr)
					done <- raddr
					return
				}
				if(ackBlock[1] == block[1] && len(data) >= 512){
					index += 512
					max = min(index + 512, len(f))
					data = f[index : max]
					block[1]++
					SendDATATo(block, data, Conn, raddr)
				}else if (ackBlock[1] == block[1] && len(data) < 512){
					//last block was acknowledged, no more need to be sent
					done <- raddr
					return
				}else{
					//do nothing, wait for correct ack or timeout
				}
			case <- time.After(1 * time.Second) :
				//resends if acknowledgment not recieved
				SendDATATo(block, data, Conn, raddr)
			case <- time.After(15 * time.Second) :
				fmt.Println("An in-progress RRQ timed out")
				done <- raddr
				return //closes out if too much time passes	
		}
	}
}

func GetRRQname(p []byte) (string, error){
  if(len(p) < 9){ //opcode + 0 + octet + 0
    return "", errors.New("badly formed RRQ packet")
  }
  index := 0
  for i := 2; i < len(p); i++ {
    if p[i] == 0 { index = i; break  }
  }
  filename := string(p[2:index])
  return filename, nil
}

func makeWRQ(filename string) ([]byte, error){
	if(len(filename) == 0){
    return nil, errors.New("no file name")
  }
  packet := []byte{0,2}
  packet = append(packet, []byte(filename)...)
  packet = append(packet, []byte{0}...)
  packet = append(packet, []byte("octet")...)
	packet = append(packet, []byte{0}...)
  return packet, nil
}

func SendWRQ(filename string, Conn *net.UDPConn){
  msg, err := makeWRQ(filename)
  checkError(err)
  _,err = Conn.Write(msg)
  checkError(err)
}

func SendWRQTo(filename string, Conn *net.UDPConn, raddr *net.UDPAddr){
  msg, err := makeWRQ(filename)
  checkError(err)
  _,err = Conn.WriteTo(msg, raddr)
  checkError(err)
}

func HandleWRQ(filename string, in <-chan []byte,  done chan<- *net.UDPAddr, Conn *net.UDPConn, raddr *net.UDPAddr, outf chan<- *File){
	SendACKTo([]byte {0, 0}, Conn, raddr) //send initial confirmation
	temp := []byte{}
	prev_block := []byte{0,0}
	for {
		select {
			case r := <- in :
				block, data, err := GetData(r)
				fmt.Println("DATA from ", raddr, " is ", len(data), " long")
				if(err != nil){
					errCode := []byte {0, 0}
					errMsg  := "badly formed data packet"
					SendERRORTo(errCode, errMsg, Conn, raddr)
					done <- raddr
					return
				}
				if(block[1] == prev_block[1]){
					//sender didn't get last ack
					SendACKTo(prev_block, Conn, raddr)
				} else {
					SendACKTo(block, Conn, raddr)
					prev_block[1] = block[1]
					temp = append(temp, data...)
					if(len(data) < 512){
						//choosing simpler option for now, could wait for
						//sender to see if they get my ACK, but also can just quit
						fmt.Println("last expected DATA from ", raddr, " recieved")
						outf <- &File{filename, temp, Conn, raddr}
						done <- raddr
						return
					}
				}
			case <-time.After(15 * time.Second) : //does this spawn new timer each for loop?
				fmt.Println("An in-progress WRQ timed out")
				done <- raddr
				return
		}
	}
}

func WriteFile(in <-chan *File, m map[string][]byte){
	for r := range in {
		_,present := m[r.Name]
		if(!present) {
			m[r.Name] = r.Data
		}else{
			errCode := []byte {0, 6}
			errMsg  := "No overwriting allowed"
			SendERRORTo(errCode, errMsg, r.Conn, r.Raddr)
		}
	}
} 

func GetWRQname(p []byte) (string, error){ 
	if(len(p) < 9){ //opcode + 0 + octet + 0
		return "", errors.New("badly formed WRQ packet")
	}
	index := 0
	for i := 2; i < len(p); i++ {
		if p[i] == 0 { index = i; break  }
	}
	filename := string(p[2:index])
	return filename, nil
}

func makeDATA(block []byte, data []byte) ([]byte, error){
  //range of block size for file: 0- 255
	if(len(data) > 512){
		return nil, errors.New("data must be 512 bytes or fewer")
	}
  packet := []byte{0,3}
  packet = append(packet, block...)
  packet = append(packet, data...)
  return packet, nil
}

func SendDATA(block []byte, data []byte, Conn *net.UDPConn){
	msg, err := makeDATA(block, data)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)
}

func SendDATATo(block []byte, data []byte, Conn *net.UDPConn, raddr *net.UDPAddr){
  msg, err := makeDATA(block, data)
  checkError(err)
  _,err = Conn.WriteTo(msg, raddr)
  checkError(err)
}


func GetData(p []byte) ([]byte, []byte, error){
	if(len(p) <= 4){
		return nil, nil, errors.New("bad DATA packet")
	}
	block := p[2:4]
	data  := p[4:]
	return block, data, nil
}

func makeACK(block []byte) ([]byte, error){
  packet := []byte{0,4}
	packet = append(packet, block...)
  return packet, nil
}

func SendACK(block []byte, Conn *net.UDPConn){
	msg, err := makeACK(block)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)	
}

func SendACKTo(block []byte, Conn *net.UDPConn, raddr *net.UDPAddr){
  msg, err := makeACK(block)
  checkError(err)
  _,err = Conn.WriteToUDP(msg, raddr)
  checkError(err)
}

func GetAckBlock(p []byte) ([]byte, error){
	if(len(p) < 4){
		return nil, errors.New("bad ACK packet")
	}
	block := p[2:4]
	return block, nil
}

func makeERROR(errCode []byte, errMsg string) ([]byte, error){
	if(len(errCode) == 0){
		return nil, errors.New("no error code")
	}
  packet := []byte{0,5}
  packet = append(packet, errCode...)
  packet = append(packet, []byte(errMsg)...)
  packet = append(packet, []byte{0}...)
  return packet, nil
}

func SendERROR(errCode []byte, errMsg string, Conn *net.UDPConn){
	msg, err := makeERROR(errCode, errMsg)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)
}

func SendERRORTo(errCode []byte, errMsg string, Conn *net.UDPConn, raddr *net.UDPAddr){
  msg, err := makeERROR(errCode, errMsg)
  checkError(err)
  _,err = Conn.WriteTo(msg, raddr)
  checkError(err)
}

func GetError(p []byte) ([]byte, []byte, error){
	if(len(p) < 5){
		return nil, nil, errors.New("bad ERROR packet")
	}
	errCode:= p[2:4]
	errMsg := p[4:len(p)]
	return errCode, errMsg, nil
}
