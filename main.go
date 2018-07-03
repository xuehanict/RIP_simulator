package main

import (
	"fmt"
	"sync"
	"bufio"
	"os"
)

type RouterID int

type router struct {
	ID           RouterID
	RouteTable   routeTable
	Update       chan *update
	GlobalUpdate chan *update
	quit         chan int
}

type routeTable struct {
	Entries []routeTableEntry
}

type routeTableEntry struct {
	TargetId RouterID
	NextHop  RouterID
	Distance int
	//TODO(xuehan): add bindWidth and link info
}

type update struct {
	sourceRouterID RouterID
	targetRouterID RouterID
	*routeTableEntry
}

func (r *router) sendUpdate(u *update) error {
	r.GlobalUpdate <- u
	return nil
}

func (r *router) handleUpdate(u *update) error {
	targetID := u.TargetId
	nextHop := u.sourceRouterID
	distance := u.Distance + 1

	if distance >= 16 {
		distance = 16
	}

	isExist := false
	for _, entry := range r.RouteTable.Entries {
		// We find the entry, so just update it.
		if entry.TargetId == targetID {
			isExist = true
			if entry.NextHop == nextHop {
				entry.Distance = distance
			} else {
				if entry.Distance > distance {
					entry.NextHop = nextHop
					entry.Distance = distance
				}
			}
		}
	}
	// if this entry is not in the route table, we add this into table.
	if !isExist {
		newEntry := routeTableEntry{
			TargetId: targetID,
			NextHop:  nextHop,
			Distance: distance,
		}
		r.RouteTable.Entries = append(r.RouteTable.Entries, newEntry)
	}

	// Broadcast this update to his neighbours
	for _, entry := range r.RouteTable.Entries {
		if entry.NextHop == entry.TargetId &&
			entry.Distance == 1 &&
			entry.TargetId != u.sourceRouterID {

			update := &update{
				sourceRouterID: r.ID,
				targetRouterID: entry.TargetId,
				routeTableEntry: &routeTableEntry{
					TargetId: targetID,
					Distance: distance,
				},
			}
			r.sendUpdate(update)
		}
	}

	return nil
}

func (r *router) start() {
	for {
		select {
		case update := <-r.Update:
			r.handleUpdate(update)
		case <-r.quit:
			fmt.Println("router %d is closed.", r.ID)
			return
		}
	}
}

func (r *router) stop() {
	close(r.quit)
}

func newRouter(id RouterID, globalUpdate chan *update) *router{
	newRouter := &router{
		ID:           id,
		Update:       make(chan *update),
		RouteTable:   routeTable{},
		GlobalUpdate: globalUpdate,
		quit:		  make(chan int),
	}
	return newRouter
}


func main() {
	wg := sync.WaitGroup{}
	quit := make(chan int)
	globalUpdateChan := make(chan *update, 10)
	routerMap := make(map[RouterID]*router)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case update := <-globalUpdateChan:
				router := routerMap[update.targetRouterID]
				router.Update <- update
			case <-quit:
				break
			}
		}
	}()

	fmt.Printf("RIP简易版模拟器\n" +
		"初始节点信息:\n" +
		"1 <-----> 2 <----> 3 <-----> 4 \n" +
		" |                         |    \n" +
		" |-----------> 5 <---------| \n ")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("输入1, 打印路由表" +
			"输入2,添加节点" +
			"输入3,添加链路" +
			"输入其他无效作废")

	}
}
