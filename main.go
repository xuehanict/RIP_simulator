package main

import (
	"fmt"
	"sync"
	"os"
	"os/signal"
	"syscall"
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
	ifUpdate := false
	isExist := false
	for _, entry := range r.RouteTable.Entries {
		// We find the entry, so just update it.
		if entry.TargetId == targetID {
			isExist = true
			if entry.NextHop == nextHop {
				entry.Distance = distance
				ifUpdate = true
			} else {
				if entry.Distance > distance {
					entry.NextHop = nextHop
					entry.Distance = distance
					ifUpdate = true
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
		ifUpdate = true
	}

	// Broadcast this update to his neighbours
	if ifUpdate {
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
	}
	return nil
}

func (r *router) start(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case update := <-r.Update:
			r.handleUpdate(update)
		case <-r.quit:
			fmt.Printf("router %d is closed.\n", r.ID)
			fmt.Println("-1")
			return
		}
	}
}

func (r *router) stop() {
	close(r.quit)
}

func newRouter(id RouterID, globalUpdate chan *update) *router {
	newRouter := &router{
		ID:           id,
		Update:       make(chan *update),
		RouteTable:   routeTable{},
		GlobalUpdate: globalUpdate,
		quit:         make(chan int),
	}
	return newRouter
}

func main() {
	wg := sync.WaitGroup{}
	quit := make(chan int)
	globalUpdateChan := make(chan *update, 10)
	routerMap := make(map[RouterID]*router)

	wg.Add(1)
	fmt.Println("update 守护线程启动 +1")
	go func() {
		defer wg.Done()
		for {
			select {
			case update := <-globalUpdateChan:
				router := routerMap[update.targetRouterID]
				router.Update <- update
			case <-quit:
				fmt.Println("update 守护线程结束 -1")
				return
			}
		}
	}()

//	wg.Add(1)
//	fmt.Println("信号监听 +1")
	go func() {
//		defer wg.Done()
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGHUP,
			syscall.SIGINT, syscall.SIGTERM,
			syscall.SIGQUIT)
		<- c
		fmt.Println("收到退出信号，开始退出")
		for _, router := range routerMap{
			router.quit <- 0
		}
		quit <- 0
		wg.Wait()
		os.Exit(0)
		//fmt.Println("信号监听结束 -1")
	}()

	fmt.Printf("RIP简易版模拟器\n" +
		"初始节点信息:\n" +
		"1 <-----> 2 <----> 3 <-----> 4 \n")

	r1 := newRouter(1, globalUpdateChan)
	r2 := newRouter(2, globalUpdateChan)
	r3 := newRouter(3, globalUpdateChan)
	r4 := newRouter(4, globalUpdateChan)

	r1.RouteTable.Entries = []routeTableEntry{
		{
			TargetId: 2,
			NextHop:  2,
			Distance: 1,
		},
		{
			TargetId: 3,
			NextHop:  2,
			Distance: 2,
		},
		{
			TargetId: 4,
			NextHop:  2,
			Distance: 3,
		},
	}
	r2.RouteTable.Entries = []routeTableEntry{
		{
			TargetId: 1,
			NextHop:  1,
			Distance: 1,
		},
		{
			TargetId: 3,
			NextHop:  3,
			Distance: 1,
		},
		{
			TargetId: 4,
			NextHop:  3,
			Distance: 2,
		},
	}
	r3.RouteTable.Entries = []routeTableEntry{
		{
			TargetId: 1,
			NextHop:  2,
			Distance: 2,
		},
		{
			TargetId: 2,
			NextHop:  2,
			Distance: 1,
		},
		{
			TargetId: 4,
			NextHop:  4,
			Distance: 1,
		},
	}
	r4.RouteTable.Entries = []routeTableEntry{
		{
			TargetId: 1,
			NextHop:  3,
			Distance: 3,
		},
		{
			TargetId: 2,
			NextHop:  3,
			Distance: 2,
		},
		{
			TargetId: 3,
			NextHop:  3,
			Distance: 1,
		},
	}
	routerMap[1] = r1
	routerMap[2] = r2
	routerMap[3] = r3
	routerMap[4] = r4

	wg.Add(4)
	fmt.Println("路由器启动 +4")

	go r1.start(&wg)
	go r2.start(&wg)
	go r3.start(&wg)
	go r4.start(&wg)

	//reader := bufio.NewReader(os.Stdin)
	var input int
	for {
		fmt.Print(
			"输入1, 打印路由表" +
				"输入2,添加节点" +
				"输入3,添加链路" +
				"输入其他无效作废\n")

		//input, err := reader.ReadString('\n')
		fmt.Scanf("%d", &input)
		switch input {
		case 1:
			for _, router := range routerMap {
				fmt.Printf("Router %d table is :\n", router.ID)
				fmt.Print(
					"Destination     nextHop    distance\n" +
						"----------------------\n")
				for _, entry := range router.RouteTable.Entries {
					fmt.Printf("%d     %d    %d\n", entry.TargetId,
						entry.NextHop, entry.Distance)
				}
			}
			break
		case 2:
			var id RouterID
			fmt.Scanf("%d", &id)
			_, ok := routerMap[id]
			if ok {
				fmt.Println("this id is exist ,please reinput")
				break
			}
			router := newRouter(id, globalUpdateChan)
			routerMap[id] = router

			wg.Add(1)
			go router.start(&wg)
			break
		case 3:
			var source, dest RouterID
			fmt.Scanf("%d,%d", &source, &dest)
			sourceRouter, ok := routerMap[source]
			if !ok {
				fmt.Println("cann't find the router")
			}
			destRouter, ok := routerMap[dest]
			if !ok {
				fmt.Println("cann't find the router")
			}
			// 因为添加了水平分割，所以两头都发
			sourceRouter.sendUpdate(&update{
				targetRouterID: dest,
				sourceRouterID: source,
				routeTableEntry: &routeTableEntry{
					Distance: 0,
					NextHop:  dest,
					TargetId: dest,
				},
			})
			destRouter.sendUpdate(&update{
				targetRouterID: source,
				sourceRouterID: dest,
				routeTableEntry: &routeTableEntry{
					Distance: 0,
					NextHop:  source,
					TargetId: source,
				},
			})
			// TODO(xuehan): check the link if exist
		default:
			fmt.Printf("Error input, please reinput")
			break
		}
	}
	// wg.Wait()
}
