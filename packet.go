package packet

import (
	"net"
	"errors"
	"fmt"
)

func checkError(err error){
	if (err != nil) {
		fmt.Println("Error: ", err)
	}
}

func makeRRQ(filename string) ([]byte, error ){
	packet, err := makeWRQ(filename)
	if(packet != nil){ 
		packet[1] = 1;
	}
	return packet, err
}

func SendRRQ(filename string, Conn *net.UDPConn){
	msg, err := makeRRQ(filename)
	checkError(err);
	_,err = Conn.Write(msg)
	checkError(err);
}

SendRRQTo(filename string, Conn *net.UDPConn){
  msg, err := makeRRQ(filename)
  checkError(err);
  _,err = Conn.WriteTo(msg)
  checkError(err);
}

func makeWRQ(filename string) ([]byte, error){
	if(len(filename) == 0){
    return nil, errors.New("no file name")
  }
  packet := []byte{0,2}
  packet = append(packet, []byte(filename)...)
  packet = append(packet, []byte{0}...)
  packet = append(packet, []byte("octet0")...)
  return packet, nil
}

func SendWRQ(filename string, Conn *net.UDPConn){
  msg, err := makeWRQ(filename)
  checkError(err)
  _,err = Conn.Write(msg)
  checkError(err)
}

SendWRQTo(filename string, Conn *net.UDPConn){
  msg, err := makeWRQ(filename)
  checkError(err)
  _,err = Conn.WriteTo(msg)
  checkError(err)
}

func makeDATA(block byte, data []byte) ([]byte, error){
  //range of block size for file: 0- 255
	if(len(data) > 512){
		return nil, errors.New("data must be 512 bytes or fewer")
	}
  packet := []byte{0,3}
  packet = append(packet, []byte{block}...)
  packet = append(packet, data...)
  return packet, nil
}

func SendDATA(block byte, data []byte, Conn *net.UDPConn){
	msg, err := makeDATA(block, data)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)
}

SendDATATo(block byte, data []byte, Conn *net.UDPConn){
  msg, err := makeDATA(block, data)
  checkError(err)
  _,err = Conn.WriteTo(msg)
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

func makeACK(block byte) ([]byte, error){
  packet := []byte{0,4,block}
  return packet, nil
}

func SendACK(block byte, Conn *net.UDPConn){
	msg, err := makeACK(block)
	checkError(err)
	_,err = Conn.Write(msg)
	checkError(err)	
}

SendACKTo(block byte, Conn *net.UDPConn){
  msg, err := makeACK(block)
  checkError(err)
  _,err = Conn.WriteTo(msg)
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

func SendERRORTo(errCode []byte, errMsg string, Conn *net.UDPConn){
  msg, err := makeERROR(errCode, errMsg)
  checkError(err)
  _,err = Conn.WriteTo(msg)
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
