import {hello} from "./hello"

export const workflowSettings = {
    id: 'tokenGen',
    trigger: 'onTokenGeneration',
    bindings:{
        "console": {},
        "kinde.fetch": {},
        "kinde.idToken": {
            resetClaims: true
        },
        "kinde.accessToken": {
            resetClaims: true
        }
    }
};

export default async function handle (event: any) {
        kinde.idToken.setCustomClaim('random', 'test');
        kinde.accessToken.setCustomClaim('test2', {"test2": hello});
        console.log('logging from action', {"balh": "blah"});
        await kinde.fetch("http://google.com");
        console.error('error log');
        return 'testing return';
}