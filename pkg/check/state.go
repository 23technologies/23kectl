package check

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Result struct {
	IsError    bool
	IsOkay     bool
	Status     string
	Hint       string
	Conditions []metav1.Condition
}
