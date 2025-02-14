// Copyright 2025 Brandon Epperson
// SPDX-License-Identifier: Apache-2.0

package roxi

import (
	"net/http"
)

// TODO: improve default 404 page.
var notFoundPage = `
<html>
    <head>
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>
            404 Page Not Found
        </title>
        <link rel="preconnect" href="https://fonts.googleapis.com">
        <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
        <link href="https://fonts.googleapis.com/css2?family=Roboto+Mono:ital,wght@0,100..700;1,100..700&display=swap" rel="stylesheet">
    </head>
    <style>
        html{
            background-color: #e74c3c;

            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
        }

        body{
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fefefe;
        }

        .error-container{
            display: block;
            vertical-align: middle;
        }
    </style>
</html>

<body>
    <div class="error-container">
        <h1>404</h1> 
        <p>
            The resource you have requested could not be found. If you are not the site owner, please contact the adminstrators for further guidance.
        </p>
    </div>
</body>
`

type notFound struct{}

func (r notFound) Encode() ([]byte, string, error) {
	return []byte(notFoundPage), "text/html", nil
}

func (r notFound) StatusCode() int {
	return http.StatusNotFound
}
