package smartag

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// SwitchCloudBoxHaState invokes the smartag.SwitchCloudBoxHaState API synchronously
// api document: https://help.aliyun.com/api/smartag/switchcloudboxhastate.html
func (client *Client) SwitchCloudBoxHaState(request *SwitchCloudBoxHaStateRequest) (response *SwitchCloudBoxHaStateResponse, err error) {
	response = CreateSwitchCloudBoxHaStateResponse()
	err = client.DoAction(request, response)
	return
}

// SwitchCloudBoxHaStateWithChan invokes the smartag.SwitchCloudBoxHaState API asynchronously
// api document: https://help.aliyun.com/api/smartag/switchcloudboxhastate.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SwitchCloudBoxHaStateWithChan(request *SwitchCloudBoxHaStateRequest) (<-chan *SwitchCloudBoxHaStateResponse, <-chan error) {
	responseChan := make(chan *SwitchCloudBoxHaStateResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.SwitchCloudBoxHaState(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// SwitchCloudBoxHaStateWithCallback invokes the smartag.SwitchCloudBoxHaState API asynchronously
// api document: https://help.aliyun.com/api/smartag/switchcloudboxhastate.html
// asynchronous document: https://help.aliyun.com/document_detail/66220.html
func (client *Client) SwitchCloudBoxHaStateWithCallback(request *SwitchCloudBoxHaStateRequest, callback func(response *SwitchCloudBoxHaStateResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *SwitchCloudBoxHaStateResponse
		var err error
		defer close(result)
		response, err = client.SwitchCloudBoxHaState(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// SwitchCloudBoxHaStateRequest is the request struct for api SwitchCloudBoxHaState
type SwitchCloudBoxHaStateRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	SmartAGId            string           `position:"Query" name:"SmartAGId"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// SwitchCloudBoxHaStateResponse is the response struct for api SwitchCloudBoxHaState
type SwitchCloudBoxHaStateResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateSwitchCloudBoxHaStateRequest creates a request to invoke SwitchCloudBoxHaState API
func CreateSwitchCloudBoxHaStateRequest() (request *SwitchCloudBoxHaStateRequest) {
	request = &SwitchCloudBoxHaStateRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Smartag", "2018-03-13", "SwitchCloudBoxHaState", "smartag", "openAPI")
	return
}

// CreateSwitchCloudBoxHaStateResponse creates a response to parse from SwitchCloudBoxHaState response
func CreateSwitchCloudBoxHaStateResponse() (response *SwitchCloudBoxHaStateResponse) {
	response = &SwitchCloudBoxHaStateResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
