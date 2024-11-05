import {hello} from "./hello"

export const workflowSettings = {
    id: 'tokenGen',
    trigger: 'onTokenGeneration',
    bindings:{
        "console": {},
        "url": {},
        "util": {},
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
    const ps = new URLSearchParams({"aaa": "bbb", "ccc": "ddd"});
    console.log('logging url search params', ps.toString());
    if (ps.toString() !== 'aaa=bbb&ccc=ddd') {
        throw new Error('URLSearchParams failed');
    }

    var txt = 'Congratulate %s on his %dth birthday!';
    var result = util.format(txt, 'Ev', 42);
    if (result !== 'Congratulate Ev on his 42th birthday!') {
        throw new Error('util.format failed');
    }
    console.log('logging with util formatting', result);

    kinde.idToken.setCustomClaim('random', 'test');
    kinde.accessToken.setCustomClaim('test2', {"test2": hello});
    console.log('logging from action', {"balh": "blah"});
    await kinde.fetch("http://google.com");
    console.error('error log');
    return 'testing return';
}