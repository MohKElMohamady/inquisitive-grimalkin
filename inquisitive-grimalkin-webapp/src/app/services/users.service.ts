import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { User } from '../models/user';

@Injectable({
  providedIn: 'root'
})
export class UsersService {

  private usersApiUrl = `localhost:8080/users/`;

  constructor(private http : HttpClient) { }

  public follow(userToBeFollowed : string) : Observable<string> {
    const followUrl = this.usersApiUrl + userToBeFollowed + `/follow`
    return this.http.post<string>(followUrl, null)
  }

  public unfollow(userToBeUnfollowed : string) : Observable<string> {
    const followUrl = this.usersApiUrl + userToBeUnfollowed + `/unfollow`;
    return this.http.post<string>(followUrl, null)
  }

  public searchForUser(username : string) : Observable<User> {
    const searchUrl = this.usersApiUrl + `search/` + username;
    return this.http.get<User>(searchUrl) 
  }
}
