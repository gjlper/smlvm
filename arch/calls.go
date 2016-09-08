package arch

import (
	"container/list"

	"e8vm.io/e8vm/arch/vpc"
)

const (
	callsControl = 0x0
	callsService = 0x4

	callsRequestAddr  = 0x8
	callsRequestLen   = 0xc
	callsResponseAddr = 0x10
	callsResponseSize = 0x14 // buffer size
	callsResponseCode = 0x18
	callsResponseLen  = 0x1c

	callsSize = 0x20
)

type calls struct {
	p        *pageOffset
	mem      *phyMemory
	services map[uint32]vpc.Service
	enabled  map[uint32]bool
	queue    *list.List
}

func newCalls(p *page, mem *phyMemory) *calls {
	return &calls{
		p:        &pageOffset{p, 0},
		mem:      mem,
		services: make(map[uint32]vpc.Service),
		queue:    list.New(),
	}
}

func (c *calls) sender(id uint32) vpc.Sender {
	return &callsSender{service: id, queue: c.queue}
}

func (c *calls) register(id uint32, s vpc.Service) {
	if id == 0 {
		panic("cannot register service 0")
	}

	c.services[id] = s
}

func (c *calls) callControl(ctrl uint8, req []byte) ([]byte, int32, *Excep) {
	switch ctrl {
	case 1: // poll message
		if c.queue.Len() == 0 {
			return nil, vpc.ErrInternal, errSleep // we will execute again
		}

		m := c.queue.Front().Value.(*callsMessage)
		c.p.writeWord(callsService, m.service) // overwrite the service
		return m.p, 0, nil

	// TODO(lonliu): add other stuff
	case 2: // list services
	case 3: // enable/disalbe service message
	}

	return nil, vpc.ErrInvalidArg, nil
}

func (c *calls) call(ctrl uint8, s uint32, req []byte) ([]byte, int32, *Excep) {
	if s == 0 {
		return c.callControl(ctrl, req)
	}

	service, found := c.services[s]
	if !found {
		return nil, vpc.ErrNotFound, nil
	}
	resp, ret := service.Handle(req)
	return resp, ret, nil
}

func (c *calls) respondCode(code int32) {
	c.p.writeWord(callsResponseCode, uint32(code))
}

func (c *calls) Tick() { c.invoke() }

func (c *calls) invoke() *Excep {
	control := c.p.readByte(callsControl)
	if control == 0 {
		return nil
	}

	service := c.p.readWord(callsService)
	reqAddr := c.p.readWord(callsRequestAddr)
	reqLen := c.p.readWord(callsRequestLen)

	var req []byte
	if reqLen > 0 {
		req = make([]byte, reqLen)
	}

	for i := range req {
		var exp *Excep
		req[i], exp = c.mem.ReadByte(reqAddr + uint32(i))
		if exp != nil {
			return exp
		}
	}

	resp, code, exp := c.call(control, service, req)
	if exp != nil {
		return exp
	}
	if code != 0 {
		c.respondCode(code)
		return nil
	}

	respAddr := c.p.readWord(callsResponseAddr)
	respSize := c.p.readWord(callsResponseSize)
	respLen := len(resp)
	if respLen > vpc.MaxLen {
		c.respondCode(vpc.ErrInternal)
		return nil
	}

	// we will write the response length anyways
	c.p.writeWord(callsResponseLen, uint32(respLen))
	if uint32(respLen) > respSize {
		c.respondCode(vpc.ErrSmallBuf)
		return nil
	}

	if resp != nil {
		for i := range resp {
			if exp := c.mem.WriteByte(respAddr+uint32(i), resp[i]); exp != nil {
				return exp
			}
		}
	}

	c.respondCode(0)
	c.p.writeByte(callsControl, 0)
	return nil
}
