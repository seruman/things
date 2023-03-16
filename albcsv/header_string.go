// Code generated by "stringer -type header"; DO NOT EDIT.

package main

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[type_-1]
	_ = x[time-2]
	_ = x[elb-3]
	_ = x[client_ip-4]
	_ = x[client_port-5]
	_ = x[target_ip-6]
	_ = x[target_port-7]
	_ = x[request_processing_time-8]
	_ = x[target_processing_time-9]
	_ = x[response_processing_time-10]
	_ = x[elb_status_code-11]
	_ = x[target_status_code-12]
	_ = x[received_bytes-13]
	_ = x[sent_bytes-14]
	_ = x[request_verb-15]
	_ = x[request_url-16]
	_ = x[request_proto-17]
	_ = x[user_agent-18]
	_ = x[ssl_cipher-19]
	_ = x[ssl_protocol-20]
	_ = x[target_group_arn-21]
	_ = x[trace_id-22]
	_ = x[domain_name-23]
	_ = x[chosen_cert_arn-24]
	_ = x[matched_rule_priority-25]
	_ = x[request_creation_time-26]
	_ = x[actions_executed-27]
	_ = x[redirect_url-28]
	_ = x[lambda_error_reason-29]
	_ = x[target_port_list-30]
	_ = x[target_status_code_list-31]
	_ = x[classification-32]
	_ = x[classification_reason-33]
}

const _header_name = "type_timeelbclient_ipclient_porttarget_iptarget_portrequest_processing_timetarget_processing_timeresponse_processing_timeelb_status_codetarget_status_codereceived_bytessent_bytesrequest_verbrequest_urlrequest_protouser_agentssl_cipherssl_protocoltarget_group_arntrace_iddomain_namechosen_cert_arnmatched_rule_priorityrequest_creation_timeactions_executedredirect_urllambda_error_reasontarget_port_listtarget_status_code_listclassificationclassification_reason"

var _header_index = [...]uint16{0, 5, 9, 12, 21, 32, 41, 52, 75, 97, 121, 136, 154, 168, 178, 190, 201, 214, 224, 234, 246, 262, 270, 281, 296, 317, 338, 354, 366, 385, 401, 424, 438, 459}

func (i header) String() string {
	i -= 1
	if i < 0 || i >= header(len(_header_index)-1) {
		return "header(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _header_name[_header_index[i]:_header_index[i+1]]
}