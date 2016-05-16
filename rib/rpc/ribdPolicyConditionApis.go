Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// ribdPolicyConditionApis.go
package rpc

import (
	"fmt"
	"ribd"
	"utils/policy"
)

func (m RIBDServicesHandler) CreatePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyConditioncfg: ", cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name, ConditionType: cfg.ConditionType, MatchProtocolConditionInfo: cfg.Protocol}
	matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.IpPrefix, MasklengthRange: cfg.MaskLengthRange}
	newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{Prefix: matchPrefix}
	err = m.server.GlobalPolicyEngineDB.ValidateConditionConfigCreate(newPolicy)
	if err != nil {
		logger.Err(fmt.Sprintln("PolicyEngine validation failed with err: ",err))
		return false,err
	}
	m.server.PolicyConditionCreateConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) DeletePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyConditionConfig: ", cfg.Name))
	err = m.server.GlobalPolicyEngineDB.ValidateConditionConfigDelete(policy.PolicyConditionConfig{Name:cfg.Name})
	if err != nil {
		logger.Err(fmt.Sprintln("PolicyEngine validation failed with err: ",err))
		return false,err
	}
	m.server.PolicyConditionDeleteConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) UpdatePolicyCondition(origconfig *ribd.PolicyCondition, newconfig *ribd.PolicyCondition, attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyConditionConfig:UpdatePolicyCondition: ", newconfig.Name))
	return true, err
}
func (m RIBDServicesHandler) GetPolicyConditionState(name string) (*ribd.PolicyConditionState, error) {
	logger.Info("Get state for Policy Condition")
	retState := ribd.NewPolicyConditionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyConditionState(fromIndex ribd.Int, rcount ribd.Int) (policyConditions *ribd.PolicyConditionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyConditionState"))
	ret,err := m.server.GetBulkPolicyConditionState(fromIndex,rcount,m.server.GlobalPolicyEngineDB)
	return ret, err
}
