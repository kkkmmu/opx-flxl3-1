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
package server

import (
//"fmt"
)

func (server *BFDServer) initBfdGlobalConfDefault() error {
	server.bfdGlobal.Enabled = true
	return nil
}

func (server *BFDServer) processGlobalConfig(gConf GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	wasEnabled := server.bfdGlobal.Enabled
	server.bfdGlobal.Enabled = gConf.Enable
	isEnabled := server.bfdGlobal.Enabled
	length := len(server.bfdGlobal.SessionsIdSlice)
	for i := 0; i < length; i++ {
		sessionId := server.bfdGlobal.SessionsIdSlice[i]
		session := server.bfdGlobal.Sessions[sessionId]
		if !wasEnabled && isEnabled {
			// Bfd enabled globally. Restart all the sessions
			session.StartBfdSession()
		}
		if wasEnabled && !isEnabled {
			// Bfd disabled globally. Stop all the sessions
			session.StopBfdSession()
		}
	}
}
