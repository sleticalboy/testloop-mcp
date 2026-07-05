package com.example;

public class TaskService {
    public String status(String state) {
        switch (state) {
            case "paid":
                return "closed";
            default:
                return "open";
        }
    }
}
