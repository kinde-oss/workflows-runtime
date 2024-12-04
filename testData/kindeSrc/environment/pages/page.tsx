"use server";

export const pageSettings = {

};

export function handleRequest(event: any) { //this tests that the function must be exported as default, otherwise it won't be started at runtime
	return "<html><body><h1>Page</h1></body></html>";
}
