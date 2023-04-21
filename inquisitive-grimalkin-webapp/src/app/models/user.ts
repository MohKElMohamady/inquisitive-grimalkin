export interface User {
    username : string,
    password : string,
    email : string,
    firstName : string,
    lastName : string,
    rank : Rank,
}


export enum Rank {
    NORMIE,
    MODERATOR,
    ADMIN
}