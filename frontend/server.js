const express = require("express");
const bodyParser = require('body-parser');
const axios = require("axios");
const path = require("path");
const app = express();
const port = 80;
const apiUrl = "http://localhost:3000/api/";
const baseUrl = "http://localhost/";

// getRedirect :: get the url to be redirected to with the given parameter.
async function getRedirect(param) {
	let data, error;
	try {
		let res = await axios({
			url: `${apiUrl}${param}`,
			method: 'get',
			timeout: 8000,
			headers: {
				'Content-Type': 'application/json',
			}
		});
		data = res.data;
	} catch (err) {
		console.error(err);
		error = err;
	};
	return {response: data, error: error};
};

// createRedirect :: takes the url that needs to be shortened and makes a post request to the api.
async function createRedirect(url) {
	let data, error;
	let payload = {url: url};
	try {
		let res = await axios({
			url: `${apiUrl}`,
			method: 'post',
			timeout: 8000,
			data: payload,
			headers: {
				'Content-Type': 'application/json',
			}
		});
		data = res.data;
	} catch (err) {
		console.error(err);
		error = err;
	};
	return {response: data, error: error};
};

// Set the rendering engine and the middleware.
app.set('view engine', 'pug');
app.use(bodyParser.json()); 
app.use(bodyParser.urlencoded({ extended: true }));
app.use(express.static(path.join(__dirname, 'public')));

// Base route, just renders the tamplate with the form.
app.get("/", (req, res) => {
	res.render('index', {returnUrl: false});
});

// Get the form data and perform an api request to register/create the short.
app.post('/', (req, res) => {
	let url = req.body.url;
	createRedirect(url)
		.then(data => {
			let retUrl = baseUrl + data.response.Data.Short;
			res.render('index', {returnUrl: true, shortUrl: retUrl});
		}).catch(err => {
			console.log(err);
		});
});

// Get the url parameters (provided short) and request the url to redirect to from the api.
// If the provided short is correct and not expired, redirect the user to the desired url.
app.get("/*", (req, res) => {
	getRedirect(req.params[0])
		.then(data => {
			if (data.response.Status == 200) {
				res.status(301).redirect(data.response.Data.Url);
			} else {
				res.status(404).send(data.response.Message);
			};
		}).catch(err => {
			res.status(500).send(err);
		});
});

// Start the server.
app.listen(port, () => {
	console.log(`Server listening on ${baseUrl}:${port}`);
});
