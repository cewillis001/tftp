package packet

import (
	"net"
	"errors"
	"fmt"
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

func HandleRRQ(filename string, in <-chan []byte, Conn *net.UDPConn, raddr *net.UDPAddr, f []byte) {
	index := 0
	max := min(512, len(f))
	data := f[index : max]
	block := []byte{0, 1}
	SendDATATo(block, data, Conn, raddr)

	for r := range in {
		//if timeout, resend last data
		//if ack, send new data
		ackBlock, err := GetAckBlock(r)
		checkError(err)
		if(ackBlock[1] == block[1] && len(data) >= 512){
			index += 512
			max = min(index + 512, len(f))
			data = f[index : max]
			block[1]++
			SendDATATo(block, data, Conn, raddr)
		} else if (len(data) < 512){
			break
		} else{
			//handle erroneous ackknowledgements
		}
	}	
	//hang around for ack block, if none before timeout, resend data
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

func HandleWRQ(filename string, in <-chan []byte, Conn *net.UDPConn, raddr *net.UDPAddr, outf chan<- *File){

	//TODO timeout
	SendACKTo([]byte {0, 0}, Conn, raddr) //send initial confirmation
	temp := []byte{}
	for r := range in {
		block, data, err := GetData(r)
		checkError(err)
		SendACKTo(block, Conn, raddr)
		temp = append(temp, data...)
		if(len(data) < 512){
			break
		}
	}
	outf <- &File{filename, temp, Conn, raddr}
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
	block := p[2:3]
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
	block := p[2:3]
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
	errCode:= p[2:3]
	errMsg := p[4:len(p)-1]
	return errCode, errMsg, nil
}
