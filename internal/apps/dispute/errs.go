/*
Copyright 2025 linux.do

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dispute

const (
	OrderNotFoundForDispute  = "订单不存在"
	DisputeNotFound          = "争议不存在"
	NotOrderMerchant         = "您不是该订单的商家"
	ReasonRequiredForRefusal = "拒绝退款时必须提供理由"
	DisputeTimeWindowExpired = "订单已交易完成,超过争议时间窗口,无法发起争议"
	DuplicateDispute         = "无法重复发起争议，如仍有疑问请联系商家或LINUX DO Credit 团队"
)
