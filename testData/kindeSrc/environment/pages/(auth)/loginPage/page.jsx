//this page won't be discovered, as there is a page.tsx in the same folder and the precedence is ts -> js -> tsx -> jsx
"use server";

export const pageSettings = {

};

export default function handleRequest(event) {
	return "<html><body><h1>Page</h1></body></html>";
}
