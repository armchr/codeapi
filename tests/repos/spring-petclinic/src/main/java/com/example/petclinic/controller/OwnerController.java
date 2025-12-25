package com.example.petclinic.controller;

import com.example.petclinic.dto.OwnerDto;
import com.example.petclinic.service.OwnerService;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.util.List;

/**
 * REST controller for Owner operations.
 */
@RestController
@RequestMapping("/api/owners")
public class OwnerController {

    private final OwnerService ownerService;

    public OwnerController(OwnerService ownerService) {
        this.ownerService = ownerService;
    }

    /**
     * GET /api/owners - Get all owners.
     */
    @GetMapping
    public ResponseEntity<List<OwnerDto>> getAllOwners() {
        return ResponseEntity.ok(ownerService.findAll());
    }

    /**
     * GET /api/owners/{id} - Get owner by ID.
     */
    @GetMapping("/{id}")
    public ResponseEntity<OwnerDto> getOwnerById(@PathVariable Long id) {
        return ResponseEntity.ok(ownerService.findById(id));
    }

    /**
     * GET /api/owners/search?lastName={lastName} - Search owners by last name.
     */
    @GetMapping("/search")
    public ResponseEntity<List<OwnerDto>> searchOwners(@RequestParam String lastName) {
        return ResponseEntity.ok(ownerService.findByLastName(lastName));
    }

    /**
     * GET /api/owners/stats/city/{city} - Get owner count by city.
     */
    @GetMapping("/stats/city/{city}")
    public ResponseEntity<Long> countByCity(@PathVariable String city) {
        return ResponseEntity.ok(ownerService.countByCity(city));
    }

    /**
     * POST /api/owners - Create a new owner.
     */
    @PostMapping
    public ResponseEntity<OwnerDto> createOwner(@Valid @RequestBody OwnerDto ownerDto) {
        OwnerDto created = ownerService.create(ownerDto);
        return new ResponseEntity<>(created, HttpStatus.CREATED);
    }

    /**
     * PUT /api/owners/{id} - Update an existing owner.
     */
    @PutMapping("/{id}")
    public ResponseEntity<OwnerDto> updateOwner(@PathVariable Long id,
                                                 @Valid @RequestBody OwnerDto ownerDto) {
        return ResponseEntity.ok(ownerService.update(id, ownerDto));
    }

    /**
     * DELETE /api/owners/{id} - Delete an owner.
     */
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteOwner(@PathVariable Long id) {
        ownerService.delete(id);
        return ResponseEntity.noContent().build();
    }
}
