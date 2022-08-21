package zjcgo

import (
	"strings"
)

type treeNode struct {
	name       string      //当前节点名称 /user
	child      []*treeNode //子节点
	routerName string      //完整的路径
	isEnd      bool        //判断是不是完全匹配
}

//添加节点
// /user/hell/
func (t *treeNode) Put(path string) {
	root := t //根节点
	strs := strings.Split(path, "/")
	//[ ,user,hell ]
	for index, name := range strs {
		if index == 0 {
			//如果是第一个点则是空格，退出即可
			continue
		}
		child := t.child
		isMatch := false //是否匹配
		for _, node := range child {
			if node.name == name {
				isMatch = true
				t = node //继续匹配下一个节点
				break
			}
		}
		//如果不匹配则把新的节点创建出来
		if !isMatch {
			//如果是最后一个节点
			isEnd := false
			if index == len(strs)-1 {
				isEnd = true
			}
			node := &treeNode{name: name, child: make([]*treeNode, 0), isEnd: isEnd}
			child = append(child, node)
			t.child = child
			t = node
		}
	}
	t = root //把修改后的前缀树更换回去
}

//从前缀树中取出路径
//get path: user/get/1
func (t *treeNode) Get(path string) *treeNode {
	strs := strings.Split(path, "/")
	routerName := ""
	for index, name := range strs {
		if index == 0 {
			continue
		}
		child := t.child
		isMash := false
		for _, node := range child {
			if node.name == name ||
				node.name == "*" ||
				strings.Contains(node.name, ":") {
				//匹配上了
				isMash = true
				routerName += "/" + node.name
				t = node
				node.routerName = routerName
				//如果是最后一个节点
				if index == len(strs)-1 {
					node.isEnd = true
					return node
				}
				break
			}
		}
		if !isMash {
			for _, node := range child {
				if node.name == "**" {
					routerName += "/" + node.name
					node.routerName = routerName
					return node
				}
			}
		}
	}
	//如果所有的路由都没匹配上
	return nil
}
