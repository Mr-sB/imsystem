package pb

type ResponseCode uint32

const (
	ResponseCodeSuccess       ResponseCode = 0
	ResponseCodeRenameError   ResponseCode = 101 //改名失败，名字重复
	ResponseCodeChatUserError ResponseCode = 102 //查无此人
)
