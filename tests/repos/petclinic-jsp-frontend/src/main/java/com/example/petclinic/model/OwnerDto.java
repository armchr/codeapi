package com.example.petclinic.model;

import java.util.ArrayList;
import java.util.List;

/**
 * Data Transfer Object for Owner entity.
 */
public class OwnerDto {
    private Long id;
    private String firstName;
    private String lastName;
    private String address;
    private String city;
    private String telephone;
    private String email;
    private List<PetDto> pets = new ArrayList<>();

    public OwnerDto() {}

    public OwnerDto(Long id, String firstName, String lastName, String address,
                    String city, String telephone, String email) {
        this.id = id;
        this.firstName = firstName;
        this.lastName = lastName;
        this.address = address;
        this.city = city;
        this.telephone = telephone;
        this.email = email;
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getFirstName() { return firstName; }
    public void setFirstName(String firstName) { this.firstName = firstName; }

    public String getLastName() { return lastName; }
    public void setLastName(String lastName) { this.lastName = lastName; }

    public String getFullName() {
        return (firstName != null ? firstName : "") + " " + (lastName != null ? lastName : "");
    }

    public String getAddress() { return address; }
    public void setAddress(String address) { this.address = address; }

    public String getCity() { return city; }
    public void setCity(String city) { this.city = city; }

    public String getTelephone() { return telephone; }
    public void setTelephone(String telephone) { this.telephone = telephone; }

    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }

    public List<PetDto> getPets() { return pets; }
    public void setPets(List<PetDto> pets) { this.pets = pets != null ? pets : new ArrayList<>(); }
}
