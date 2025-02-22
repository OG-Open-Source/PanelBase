package utils

import "github.com/sirupsen/logrus"

func HandleError(err error, message string) {
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "error": err.Error(),
        }).Fatal(message)
    }
} 